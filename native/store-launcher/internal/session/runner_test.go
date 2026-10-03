package session

import (
	"testing"

	processes "github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/process"
)

func TestActivatedProcessSelectsReturnedPID(t *testing.T) {
	snapshot := []processes.Info{
		{PID: 10, Name: "unrelated.exe", CreationTime: 100},
		{PID: 20, Name: "Game.exe", CreationTime: 200},
	}
	match, found := activatedProcess(snapshot, 20)
	if !found || match.Process.PID != 20 {
		t.Fatalf("activatedProcess = %+v, %t", match, found)
	}
	if _, found := activatedProcess(snapshot, 30); found {
		t.Fatal("activatedProcess selected an absent PID")
	}
}
