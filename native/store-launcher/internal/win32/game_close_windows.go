//go:build windows

package win32

import (
	"fmt"
	"strconv"
	"time"

	processes "github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/process"
)

// GameCloser acts only on the exact process identity recorded before the
// helper died. A normal close request gets a grace period before force.
type GameCloser struct{ Source processes.Source }

func (c GameCloser) CloseGame(target processes.Info, grace, poll time.Duration) (bool, error) {
	matching, err := c.sameProcess(target)
	if err != nil || !matching {
		return false, err
	}
	_, _ = runTaskkill("/PID", strconv.FormatUint(uint64(target.PID), 10))
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		matching, err = c.sameProcess(target)
		if err != nil || !matching {
			return false, err
		}
		time.Sleep(poll)
	}
	matching, err = c.sameProcess(target)
	if err != nil || !matching {
		return false, err
	}
	if output, killErr := runTaskkill("/F", "/PID", strconv.FormatUint(uint64(target.PID), 10)); killErr != nil {
		matching, err = c.sameProcess(target)
		if err != nil {
			return true, err
		}
		if matching {
			return true, fmt.Errorf("force close game: %w: %s", killErr, output)
		}
	}
	return true, nil
}

func (c GameCloser) sameProcess(target processes.Info) (bool, error) {
	snapshot, err := c.Source.Snapshot()
	if err != nil {
		return false, err
	}
	for _, current := range snapshot {
		if current.Identity() == target.Identity() && current.Name == target.Name {
			return true, nil
		}
	}
	return false, nil
}
