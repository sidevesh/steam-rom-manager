package epic

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/cli"
	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/logging"
	processes "github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/process"
)

type Closer interface {
	CloseEpic(processes.Info, time.Duration, time.Duration) (bool, error)
}

type Controller struct {
	Options cli.Options
	Source  processes.Source
	Closer  Closer
	Logger  *logging.Logger
	Sync    *SyncLog
}

func NewController(options cli.Options, source processes.Source, closer Closer, logger *logging.Logger) *Controller {
	return &Controller{Options: options, Source: source, Closer: closer, Logger: logger, Sync: NewSyncLog(options.EpicLogFile)}
}

func (c *Controller) BeforeLaunch() error {
	snapshot, err := c.Source.Snapshot()
	if err != nil {
		return err
	}
	if len(epicProcesses(snapshot)) == 0 {
		c.Logger.Printf("epic pre-launch: no existing launcher")
		return nil
	}
	c.Logger.Printf("epic pre-launch: existing launcher detected; waiting up to %s for active cloud sync", c.Options.EpicPreSyncWait)
	c.waitForSync("pre-launch", c.Options.EpicPreSyncWait, func() bool { return len(c.Sync.Active) == 0 })
	return c.closeLauncher("pre-launch")
}

func (c *Controller) AfterGame(endedAt time.Time) error {
	id := epicAppID(c.Options.URI)
	c.Logger.Printf("epic post-game: waiting up to %s for cloud sync app=%q", c.Options.EpicPostSyncWait, id)
	// Epic can start the upload immediately as the game exits, before process
	// discovery's handoff grace expires. Accept that small overlap.
	cutoff := endedAt.Add(-2 * time.Second).UTC()
	c.waitForSync("post-game", c.Options.EpicPostSyncWait, func() bool {
		when := c.Sync.SuccessAt[id]
		return id != "" && !when.IsZero() && !when.Before(cutoff) && len(c.Sync.Active) == 0
	})
	if failed := c.Sync.FailureAt[id]; !failed.IsZero() && !failed.Before(cutoff) {
		c.Logger.Printf("epic post-game: cloud sync reported failure app=%q; closing after configured fallback", id)
	}
	return c.closeLauncher("post-game")
}

func (c *Controller) waitForSync(phase string, limit time.Duration, done func() bool) {
	deadline := time.Now().Add(limit)
	loggedError := false
	for {
		if err := c.Sync.Poll(); err != nil && !loggedError {
			c.Logger.Printf("epic %s: cloud sync log unavailable path=%q error=%v; using timeout fallback", phase, c.Sync.Path, err)
			loggedError = true
		} else if err == nil && done() {
			c.Logger.Printf("epic %s: cloud sync condition satisfied", phase)
			return
		}
		if !time.Now().Before(deadline) {
			c.Logger.Printf("epic %s: cloud sync wait timed out after %s activeSyncs=%d; closing launcher", phase, limit, len(c.Sync.Active))
			return
		}
		pause := c.Options.PollInterval
		if remaining := time.Until(deadline); pause > remaining {
			pause = remaining
		}
		time.Sleep(pause)
	}
}

func (c *Controller) closeLauncher(phase string) error {
	snapshot, err := c.Source.Snapshot()
	if err != nil {
		return err
	}
	for _, process := range epicProcesses(snapshot) {
		c.Logger.Printf("epic %s: closing launcher pid=%d created=%d", phase, process.PID, process.CreationTime)
		forced, err := c.Closer.CloseEpic(process, c.Options.EpicExitGrace, c.Options.PollInterval)
		if err != nil {
			return fmt.Errorf("close Epic launcher pid=%d: %w", process.PID, err)
		}
		c.Logger.Printf("epic %s: launcher closed pid=%d forced=%t", phase, process.PID, forced)
	}
	return nil
}

func epicProcesses(snapshot []processes.Info) []processes.Info {
	var result []processes.Info
	for _, process := range snapshot {
		if strings.EqualFold(process.Name, "EpicGamesLauncher.exe") && process.CreationTime != 0 {
			result = append(result, process)
		}
	}
	return result
}

func epicAppID(uri string) string {
	parsed, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.Trim(parsed.Path, "/"))
}
