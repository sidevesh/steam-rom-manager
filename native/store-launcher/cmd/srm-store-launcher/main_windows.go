//go:build windows

package main

import (
	"os"
	"runtime"

	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/cli"
	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/logging"
	processes "github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/process"
	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/session"
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
	if err := session.Run(options, processes.WindowsSource{}, win32.ShellLauncher{}, logger); err != nil {
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
