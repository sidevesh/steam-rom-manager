package epic

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSyncLogReadsEpicCompletionAndAppIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "EpicGamesLauncher.log")
	start := "[2026.10.06-13.24.23:758][299]LogCloudSync: Cloud Sync: Sync Started for account:namespace:007ff8f4e30845a687e66aa77eb3e965\n"
	finish := "[2026.10.06-13.24.28:637][522]LogCloudSync: Cloud Sync: Exiting Cloud Sync - SUCCESS - AppName: account:namespace:007ff8f4e30845a687e66aa77eb3e965\n"
	if err := os.WriteFile(path, []byte(start), 0o600); err != nil {
		t.Fatal(err)
	}
	log := NewSyncLog(path)
	if err := log.Poll(); err != nil {
		t.Fatal(err)
	}
	id := "007ff8f4e30845a687e66aa77eb3e965"
	if !log.Active[id] {
		t.Fatal("cloud sync start was not detected")
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(finish); err != nil {
		t.Fatal(err)
	}
	file.Close()
	if err := log.Poll(); err != nil {
		t.Fatal(err)
	}
	if log.Active[id] || log.SuccessAt[id].IsZero() {
		t.Fatalf("completion state active=%t success=%s", log.Active[id], log.SuccessAt[id])
	}
	want := time.Date(2026, 10, 6, 13, 24, 28, 637000000, time.UTC)
	if !log.SuccessAt[id].Equal(want) {
		t.Fatalf("completion timestamp=%s, want %s", log.SuccessAt[id], want)
	}
}

func TestSyncLogIgnoresIncompleteLineUntilFinished(t *testing.T) {
	path := filepath.Join(t.TempDir(), "EpicGamesLauncher.log")
	line := "[2026.10.06-13.24.23:758][299]LogCloudSync: Cloud Sync: Sync Started for account:namespace:game\n"
	if err := os.WriteFile(path, []byte(line[:len(line)-1]), 0o600); err != nil {
		t.Fatal(err)
	}
	log := NewSyncLog(path)
	if err := log.Poll(); err != nil {
		t.Fatal(err)
	}
	if log.Active["game"] {
		t.Fatal("incomplete line was consumed")
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	file.WriteString("\n")
	file.Close()
	if err := log.Poll(); err != nil || !log.Active["game"] {
		t.Fatalf("completed line was not consumed: %v", err)
	}
}
