//go:build windows

package win32

import (
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	processes "github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/process"
	"golang.org/x/sys/windows"
)

type EpicCloser struct {
	Source processes.Source
}

func (c EpicCloser) CloseEpic(process processes.Info, grace, poll time.Duration) (bool, error) {
	stillRunning, err := c.sameProcess(process)
	if err != nil || !stillRunning {
		return false, err
	}
	// A non-forced taskkill asks the application to close. Epic may instead
	// minimize to its tray, in which case the configured grace expires.
	_, _ = runTaskkill("/PID", strconv.FormatUint(uint64(process.PID), 10))
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		stillRunning, err = c.sameProcess(process)
		if err != nil || !stillRunning {
			return false, err
		}
		pause := poll
		if pause > time.Until(deadline) {
			pause = time.Until(deadline)
		}
		time.Sleep(pause)
	}
	stillRunning, err = c.sameProcess(process)
	if err != nil || !stillRunning {
		return false, err
	}
	// Match OSOL's forced process-tree fallback only after the normal close
	// request had time to finish. Recheck PID + creation time before acting.
	if output, killErr := runTaskkill("/F", "/T", "/PID", strconv.FormatUint(uint64(process.PID), 10)); killErr != nil {
		stillRunning, err = c.sameProcess(process)
		if err != nil {
			return true, err
		}
		if stillRunning {
			return true, fmt.Errorf("taskkill failed: %w: %s", killErr, output)
		}
	}
	return true, nil
}

func (c EpicCloser) sameProcess(target processes.Info) (bool, error) {
	snapshot, err := c.Source.Snapshot()
	if err != nil {
		return false, err // Never risk terminating a reused PID when identity is unknown.
	}
	for _, process := range snapshot {
		if process.Identity() == target.Identity() && process.Name == target.Name {
			return true, nil
		}
	}
	return false, nil
}

func runTaskkill(args ...string) ([]byte, error) {
	command := exec.Command("taskkill", args...)
	// The helper is a GUI executable; a console child can otherwise flash in
	// Steam Remote Play and take focus during the two Epic close attempts.
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	return command.CombinedOutput()
}
