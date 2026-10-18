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
	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/recovery"
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
	logger.Printf("starting version=%s os=%s architecture=%s pid=%d", version, runtime.GOOS, runtime.GOARCH, os.Getpid())
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.Printf("unexpected panic: %v", recovered)
			exitCode = session.ExitInternalError
		}
		logger.Printf("process exit pid=%d code=%d", os.Getpid(), exitCode)
	}()
	stopObserving, observeError := observeCloseRequests(logger)
	if observeError != nil {
		logger.Printf("close-request observer unavailable: %v", observeError)
	} else {
		defer stopObserving()
	}
	if err != nil {
		logger.Printf("invalid arguments: %v", err)
		return session.ExitInvalidArguments
	}
	source := processes.WindowsSource{}
	logProcessIdentity(source, logger)
	if options.UseAgent {
		if err := recovery.AgentReady(); err != nil {
			logger.Printf("agent required but unavailable: %v", err)
			return session.ExitInternalError
		}
		logger.Printf("agent heartbeat verified pid=%d", os.Getpid())
	}
	hooks := session.Hooks{}
	if options.UseAgent {
		var recordPath string
		hooks.GameFound = func(game processes.Info) error {
			snapshot, err := source.Snapshot()
			if err != nil {
				return err
			}
			var helper processes.Info
			for _, current := range snapshot {
				if current.PID == uint32(os.Getpid()) {
					helper = current
					break
				}
			}
			recordPath, err = recovery.Register(recovery.Record{
				Helper: helper, Game: game, Options: options, RegisteredAt: time.Now(),
			})
			if err == nil {
				logger.Printf("agent session registered path=%q helperPID=%d helperCreated=%d gamePID=%d gameCreated=%d", recordPath, helper.PID, helper.CreationTime, game.PID, game.CreationTime)
			}
			return err
		}
		hooks.Completed = func() error {
			if err := recovery.Complete(recordPath); err != nil {
				return err
			}
			logger.Printf("agent session completed helperPID=%d", os.Getpid())
			return nil
		}
	}
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
