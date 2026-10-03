package session

import (
	"fmt"
	"time"

	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/cli"
	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/epic"
	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/logging"
	processes "github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/process"
)

const (
	ExitSuccess          = 0
	ExitInvalidArguments = 1
	ExitActivationFailed = 2
	ExitProcessNotFound  = 3
	ExitInternalError    = 8
)

type URILauncher interface {
	Open(uri string) error
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

func Run(options cli.Options, source processes.Source, launcher URILauncher, logger *logging.Logger) error {
	return RunWithClock(options, source, launcher, logger, systemClock{})
}

func RunWithClock(options cli.Options, source processes.Source, launcher URILauncher, logger *logging.Logger, clock Clock) error {
	baseline, err := source.Snapshot()
	if err != nil {
		return internalError("capture pre-launch process baseline", err)
	}
	launchTime := clock.Now()
	logger.Printf("baseline captured processes=%d timestamp=%s", len(baseline), launchTime.Format(time.RFC3339Nano))
	logger.Printf(
		"arguments uri=%q exe=%q installDir=%q process=%q startTimeout=%s pollInterval=%s handoffGrace=%s allowExisting=%t",
		epic.SanitizedURI(options.URI), options.ExpectedExecutable, options.InstallDirectory,
		options.ProcessOverride, options.StartTimeout, options.PollInterval, options.HandoffGrace,
		options.AllowExistingMatch,
	)
	if err := launcher.Open(options.URI); err != nil {
		logger.Printf("Epic URI activation failed: %v", err)
		return &ExitError{Code: ExitActivationFailed, Err: err}
	}
	logger.Printf("Epic URI activation succeeded")

	detector := NewDetector(
		options.ExpectedExecutable,
		options.ProcessOverride,
		options.InstallDirectory,
		baseline,
		launchTime,
	)
	deadline := launchTime.Add(options.StartTimeout)
	var snapshot []processes.Info
	var selected Match
	for {
		clock.Sleep(options.PollInterval)
		snapshot, err = source.Snapshot()
		if err != nil {
			return internalError("capture process snapshot during discovery", err)
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
	var emptySince time.Time
	for {
		clock.Sleep(options.PollInterval)
		snapshot, err = source.Snapshot()
		if err != nil {
			return internalError("capture process snapshot while tracking", err)
		}
		for _, event := range tracker.Update(snapshot, detector) {
			logger.Printf("%s", event)
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

func internalError(operation string, err error) error {
	return &ExitError{Code: ExitInternalError, Err: fmt.Errorf("%s: %w", operation, err)}
}
