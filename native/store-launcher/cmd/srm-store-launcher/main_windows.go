//go:build windows

package main

import (
	"os"
	"runtime"
	"time"

	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/cli"
	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/epic"
	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/logging"
	processes "github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/process"
	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/session"
	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/storeuri"
	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/win32"
)

var version = "dev"

func main() {
	os.Exit(run())
}

func run() (exitCode int) {
	options, err := cli.Parse(os.Args[1:])
	logger := logging.New(options.LogFile)
	defer logger.Close()
	logger.Printf("starting version=%s os=%s architecture=%s", version, runtime.GOOS, runtime.GOARCH)
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.Printf("unexpected panic: %v", recovered)
			exitCode = session.ExitInternalError
		}
	}()
	if err != nil {
		logger.Printf("invalid arguments: %v", err)
		return session.ExitInvalidArguments
	}
	source := processes.WindowsSource{}
	hooks := session.Hooks{}
	if options.Store == storeuri.Epic {
		controller := epic.NewController(options, source, win32.EpicCloser{Source: source}, logger)
		hooks.BeforeLaunch = controller.BeforeLaunch
		hooks.AfterGame = controller.AfterGame
		logger.Printf("epic lifecycle enabled preSyncWait=%s postSyncWait=%s exitGrace=%s", options.EpicPreSyncWait, options.EpicPostSyncWait, options.EpicExitGrace)
	}
	if err := session.RunWithHooks(options, source, win32.ShellLauncher{}, logger, realClock{}, hooks); err != nil {
		if exitError, ok := err.(*session.ExitError); ok {
			logger.Printf("exiting code=%d reason=%v", exitError.Code, exitError.Err)
			return exitError.Code
		}
		logger.Printf("exiting code=%d reason=%v", session.ExitInternalError, err)
		return session.ExitInternalError
	}
	logger.Printf("exiting code=%d", session.ExitSuccess)
	return session.ExitSuccess
}

type realClock struct{}

func (realClock) Now() time.Time               { return time.Now() }
func (realClock) Sleep(duration time.Duration) { time.Sleep(duration) }
