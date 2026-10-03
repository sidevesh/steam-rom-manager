//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/logging"
	processes "github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/process"
	"golang.org/x/sys/windows"
)

var setConsoleCtrlHandler = windows.NewLazySystemDLL("kernel32.dll").NewProc("SetConsoleCtrlHandler")

// observeCloseRequests records console control events without consuming them.
// The current GUI helper has no window to receive WM_CLOSE, and Windows does
// not notify a process before TerminateProcess kills it. A missing process-exit
// record after a heartbeat is therefore useful evidence, not a signal name.
func observeCloseRequests(logger *logging.Logger) (func(), error) {
	handler := syscall.NewCallback(func(code uint32) uintptr {
		logger.Printf("windows console close request pid=%d code=%d name=%s handled=false", os.Getpid(), code, controlEventName(code))
		return 0 // Preserve the default Windows handling of the event.
	})
	result, _, err := setConsoleCtrlHandler.Call(handler, 1)
	if result == 0 {
		return nil, fmt.Errorf("SetConsoleCtrlHandler: %w", err)
	}
	logger.Printf("close-request observer ready pid=%d source=windows-console", os.Getpid())
	return func() { _, _, _ = setConsoleCtrlHandler.Call(handler, 0) }, nil
}

func controlEventName(code uint32) string {
	switch code {
	case 0:
		return "CTRL_C_EVENT"
	case 1:
		return "CTRL_BREAK_EVENT"
	case 2:
		return "CTRL_CLOSE_EVENT"
	case 5:
		return "CTRL_LOGOFF_EVENT"
	case 6:
		return "CTRL_SHUTDOWN_EVENT"
	default:
		return "unknown"
	}
}

func logProcessIdentity(source processes.Source, logger *logging.Logger) {
	snapshot, err := source.Snapshot()
	if err != nil {
		logger.Printf("process identity unavailable pid=%d error=%v", os.Getpid(), err)
		return
	}
	pid := uint32(os.Getpid())
	var parentPID uint32
	for _, process := range snapshot {
		if process.PID == pid {
			parentPID = process.ParentPID
			break
		}
	}
	parentName := ""
	for _, process := range snapshot {
		if process.PID == parentPID {
			parentName = process.Name
			break
		}
	}
	logger.Printf("process identity pid=%d parentPID=%d parentName=%q", pid, parentPID, parentName)
}
