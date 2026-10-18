//go:build windows

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/epic"
	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/logging"
	processes "github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/process"
	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/recovery"
	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/storeuri"
	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/win32"
	"golang.org/x/sys/windows"
)

const scanInterval = 2 * time.Second

var version = "dev"

func main() {
	logger := logging.New(filepath.Join(os.Getenv("LOCALAPPDATA"), "Steam ROM Manager", "logs", "store-agent.log"))
	defer logger.Close()
	name, _ := windows.UTF16PtrFromString("Local\\SRMStoreAgent")
	mutex, err := windows.CreateMutex(nil, false, name)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		logger.Printf("agent already running in this user session")
		return
	}
	if err != nil {
		logger.Printf("agent mutex error: %v", err)
		return
	}
	defer windows.CloseHandle(mutex)
	logger.Printf("agent started version=%s pid=%d", version, os.Getpid())
	source := processes.WindowsSource{}
	for {
		if err := recovery.WriteHeartbeat(); err != nil {
			logger.Printf("agent heartbeat error: %v", err)
		}
		paths, err := recovery.Sessions()
		if err != nil {
			logger.Printf("agent session scan error: %v", err)
		} else if len(paths) != 0 {
			snapshot, err := source.Snapshot()
			if err != nil {
				logger.Printf("agent process snapshot error: %v", err)
			} else {
				for _, path := range paths {
					recoverSession(path, snapshot, source, logger)
				}
			}
		}
		time.Sleep(scanInterval)
	}
}

func recoverSession(path string, snapshot []processes.Info, source processes.Source, logger *logging.Logger) {
	record, err := recovery.Read(path)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		logger.Printf("agent record read failed path=%q error=%v", path, err)
		return
	}
	if hasIdentity(snapshot, record.Helper) {
		return
	}
	logger.Printf("agent helper missing path=%q helperPID=%d helperCreated=%d gamePID=%d gameCreated=%d age=%s", path, record.Helper.PID, record.Helper.CreationTime, record.Game.PID, record.Game.CreationTime, time.Since(record.RegisteredAt).Round(time.Second))
	if strings.HasSuffix(path, ".json") {
		claimed := strings.TrimSuffix(path, ".json") + ".recovering"
		if err := os.Rename(path, claimed); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				logger.Printf("agent record claim failed path=%q error=%v", path, err)
			}
			return
		}
		path = claimed
	}
	logger.Printf("agent recovery started helperPID=%d helperCreated=%d gamePID=%d gameCreated=%d store=%q", record.Helper.PID, record.Helper.CreationTime, record.Game.PID, record.Game.CreationTime, record.Options.Store)
	if hasIdentity(snapshot, record.Game) {
		logger.Printf("agent game close requested pid=%d name=%q grace=%s", record.Game.PID, record.Game.Name, 5*time.Second)
		closeStarted := time.Now()
		forced, err := (win32.GameCloser{Source: source}).CloseGame(record.Game, 5*time.Second, record.Options.PollInterval)
		logger.Printf("agent game close pid=%d forced=%t elapsed=%s error=%v", record.Game.PID, forced, time.Since(closeStarted).Round(time.Millisecond), err)
		if err != nil {
			return
		}
	} else {
		logger.Printf("agent game already gone pid=%d", record.Game.PID)
	}
	if record.Options.Store == storeuri.Epic {
		logger.Printf("agent Epic cleanup started postSyncWait=%s exitGrace=%s", record.Options.EpicPostSyncWait, record.Options.EpicExitGrace)
		cleanupStarted := time.Now()
		controller := epic.NewController(record.Options, source, win32.EpicCloser{Source: source}, logger)
		if err := controller.AfterGame(time.Now()); err != nil {
			logger.Printf("agent Epic cleanup failed elapsed=%s error=%v", time.Since(cleanupStarted).Round(time.Millisecond), err)
			return
		}
		logger.Printf("agent Epic cleanup completed elapsed=%s", time.Since(cleanupStarted).Round(time.Millisecond))
	}
	if err := os.Remove(path); err != nil {
		logger.Printf("agent record removal error path=%q error=%v", path, err)
		return
	}
	logger.Printf("agent recovery completed helperPID=%d gamePID=%d", record.Helper.PID, record.Game.PID)
}

func hasIdentity(snapshot []processes.Info, target processes.Info) bool {
	if target.CreationTime == 0 {
		return false
	}
	for _, current := range snapshot {
		if current.Identity() == target.Identity() && strings.EqualFold(current.Name, target.Name) {
			return true
		}
	}
	return false
}
