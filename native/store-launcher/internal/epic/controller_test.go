package epic

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/cli"
	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/logging"
	processes "github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/process"
)

type fixedSource struct{ snapshot []processes.Info }

func (s fixedSource) Snapshot() ([]processes.Info, error) { return s.snapshot, nil }

type recordingCloser struct{ closed []uint32 }

func (c *recordingCloser) CloseEpic(process processes.Info, _, _ time.Duration) (bool, error) {
	c.closed = append(c.closed, process.PID)
	return false, nil
}

func TestExistingEpicWaitFallsBackWhenLogMissing(t *testing.T) {
	closer := &recordingCloser{}
	options := cli.Options{EpicLogFile: filepath.Join(t.TempDir(), "missing.log"), EpicPreSyncWait: 0, PollInterval: time.Millisecond}
	logger := logging.New(filepath.Join(t.TempDir(), "helper.log"))
	defer logger.Close()
	controller := NewController(options, fixedSource{[]processes.Info{{PID: 41, Name: "EpicGamesLauncher.exe", CreationTime: 123}}}, closer, logger)
	if err := controller.BeforeLaunch(); err != nil {
		t.Fatal(err)
	}
	if len(closer.closed) != 1 || closer.closed[0] != 41 {
		t.Fatalf("closed PIDs = %v", closer.closed)
	}
}

func TestPostGameUsesMatchingCloudSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "EpicGamesLauncher.log")
	when := time.Now().UTC()
	line := "[" + when.Format("2006.01.02-15.04.05:000") + "][522]LogCloudSync: Cloud Sync: Exiting Cloud Sync - SUCCESS - AppName: account:namespace:foo\n"
	if err := os.WriteFile(path, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	closer := &recordingCloser{}
	options := cli.Options{URI: "com.epicgames.launcher://apps/foo?action=launch", EpicLogFile: path, EpicPostSyncWait: time.Second, PollInterval: time.Millisecond}
	logger := logging.New(filepath.Join(t.TempDir(), "helper.log"))
	defer logger.Close()
	controller := NewController(options, fixedSource{[]processes.Info{{PID: 42, Name: "EpicGamesLauncher.exe", CreationTime: 123}}}, closer, logger)
	start := time.Now()
	if err := controller.AfterGame(when.Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 500*time.Millisecond || len(closer.closed) != 1 {
		t.Fatalf("cloud success was not acted on promptly; elapsed=%s closed=%v", time.Since(start), closer.closed)
	}
}
