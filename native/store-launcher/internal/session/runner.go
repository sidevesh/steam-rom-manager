package session

import (
	"fmt"
	"os"
	"time"

	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/cli"
	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/logging"
	processes "github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/process"
	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/storeuri"
)

const (
	ExitSuccess          = 0
	ExitInvalidArguments = 1
	ExitActivationFailed = 2
	ExitProcessNotFound  = 3
	ExitInternalError    = 8
)

type Launcher interface {
	Launch(cli.Options) (uint32, error)
}

type Clock interface {
	Now() time.Time
	Sleep(time.Duration)
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}

func (systemClock) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	return e.Err.Error()
}

func Run(options cli.Options, source processes.Source, launcher Launcher, logger *logging.Logger) error {
	return RunWithClock(options, source, launcher, logger, systemClock{})
}

func RunWithClock(options cli.Options, source processes.Source, launcher Launcher, logger *logging.Logger, clock Clock) error {
	baseline, err := source.Snapshot()
	if err != nil {
		return internalError("capture pre-launch process baseline", err)
	}
	launchTime := clock.Now()
	logger.Printf("baseline captured processes=%d timestamp=%s", len(baseline), launchTime.Format(time.RFC3339Nano))
	logger.Printf(
		"arguments store=%q uri=%q launchExe=%q launchArgCount=%d launchCwd=%q aumid=%q exe=%q installDir=%q process=%q startTimeout=%s pollInterval=%s handoffGrace=%s allowExisting=%t",
		options.Store, sanitizedURI(options.URI), options.LaunchExecutable, len(options.LaunchArguments), options.LaunchDirectory, options.AppUserModelID, options.ExpectedExecutable, options.InstallDirectory,
		options.ProcessOverride, options.StartTimeout, options.PollInterval, options.HandoffGrace,
		options.AllowExistingMatch,
	)
	if options.ExpectedExecutable != "" {
		_, statError := os.Stat(options.ExpectedExecutable)
		logger.Printf("expected executable check path=%q exists=%t error=%v", options.ExpectedExecutable, statError == nil, statError)
	}
	if options.InstallDirectory != "" {
		_, statError := os.Stat(options.InstallDirectory)
		logger.Printf("install directory check path=%q exists=%t error=%v", options.InstallDirectory, statError == nil, statError)
	}
	activatedPID, err := launcher.Launch(options)
	if err != nil {
		logger.Printf("store activation failed store=%q: %v", options.Store, err)
		return &ExitError{Code: ExitActivationFailed, Err: err}
	}
	logger.Printf("store activation succeeded store=%q activatedPID=%d", options.Store, activatedPID)

	detector := NewDetector(
		options.ExpectedExecutable,
		options.ProcessOverride,
		options.InstallDirectory,
		baseline,
		launchTime,
	)
	deadline := launchTime.Add(options.StartTimeout)
	lastDiscoveryLog := launchTime
	var snapshot []processes.Info
	var selected Match
	for {
		clock.Sleep(options.PollInterval)
		snapshot, err = source.Snapshot()
		if err != nil {
			return internalError("capture process snapshot during discovery", err)
		}
		if clock.Now().Sub(lastDiscoveryLog) >= 5*time.Second {
			logger.Printf("discovery pending elapsed=%s remaining=%s processCount=%d", clock.Now().Sub(launchTime), deadline.Sub(clock.Now()), len(snapshot))
			lastDiscoveryLog = clock.Now()
		}
		if match, found := activatedProcess(snapshot, activatedPID); found {
			selected = match
			logger.Printf("selected activated pid=%d name=%q executable=%q", match.Process.PID, match.Process.Name, match.Process.Executable)
			break
		}
		if match, found := detector.Select(snapshot, options.AllowExistingMatch); found {
			selected = match
			logger.Printf("selected root pid=%d name=%q executable=%q reason=%s", match.Process.PID, match.Process.Name, match.Process.Executable, match.Reason)
			break
		}
		if !clock.Now().Before(deadline) {
			for rank, process := range detector.TimeoutCandidates(snapshot, 10) {
				logger.Printf("timeout candidate rank=%d pid=%d name=%q executable=%q created=%d", rank+1, process.PID, process.Name, process.Executable, process.CreationTime)
			}
			logger.Printf("game process discovery timed out")
			return &ExitError{Code: ExitProcessNotFound, Err: fmt.Errorf("game process not found before timeout")}
		}
	}

	tracker := NewTracker(selected.Process)
	logger.Printf("tracking started rootPID=%d rootName=%q rootExecutable=%q", selected.Process.PID, selected.Process.Name, selected.Process.Executable)
	var emptySince time.Time
	lastTrackingLog := clock.Now()
	for {
		clock.Sleep(options.PollInterval)
		snapshot, err = source.Snapshot()
		if err != nil {
			return internalError("capture process snapshot while tracking", err)
		}
		for _, event := range tracker.Update(snapshot, detector) {
			logger.Printf("%s", event)
		}
		if clock.Now().Sub(lastTrackingLog) >= 15*time.Second {
			logger.Printf("tracking heartbeat elapsed=%s trackedProcessCount=%d snapshotProcessCount=%d", clock.Now().Sub(launchTime), tracker.Count(), len(snapshot))
			lastTrackingLog = clock.Now()
		}
		if !tracker.Empty() {
			emptySince = time.Time{}
			continue
		}

		if match, found := detector.Select(snapshot, false); found {
			tracker.Add(match.Process)
			emptySince = time.Time{}
			logger.Printf("handoff selected pid=%d name=%q executable=%q reason=%s", match.Process.PID, match.Process.Name, match.Process.Executable, match.Reason)
			continue
		}
		if emptySince.IsZero() {
			emptySince = clock.Now()
			logger.Printf("tracked set empty; entering handoff grace period")
		}
		if clock.Now().Sub(emptySince) >= options.HandoffGrace {
			logger.Printf("game session completed; handoff grace expired")
			return nil
		}
	}
}

func activatedProcess(snapshot []processes.Info, pid uint32) (Match, bool) {
	if pid == 0 {
		return Match{}, false
	}
	for _, process := range snapshot {
		if process.PID == pid && process.CreationTime != 0 {
			return Match{Process: process, Reason: "Windows app activation process ID"}, true
		}
	}
	return Match{}, false
}

func sanitizedURI(uri string) string {
	if uri == "" {
		return ""
	}
	return storeuri.Sanitized(uri)
}

func internalError(operation string, err error) error {
	return &ExitError{Code: ExitInternalError, Err: fmt.Errorf("%s: %w", operation, err)}
}
