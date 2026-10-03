package session

import (
	"testing"
	"time"

	processes "github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/process"
)

func TestTrackerKeepsOnlyGameProcesses(t *testing.T) {
	root := processes.Info{PID: 10, Name: "Prey.exe", Executable: `C:\Games\Prey\Prey.exe`, CreationTime: 200}
	gameChild := processes.Info{PID: 11, ParentPID: 10, Name: "Game.exe", Executable: `C:\Games\Prey\Binaries\Game.exe`, CreationTime: 300}
	overlay := processes.Info{PID: 12, ParentPID: 10, Name: "Overlay.exe", Executable: `C:\Program Files\Epic\Overlay.exe`, CreationTime: 300}
	staleParent := processes.Info{PID: 13, ParentPID: 10, Name: "Other.exe", Executable: `C:\Games\Prey\Other.exe`, CreationTime: 100}
	detector := NewDetector(root.Executable, "", `C:\Games\Prey`, nil, time.Unix(0, 150))
	tracker := NewTracker(root)

	tracker.Update([]processes.Info{root, gameChild, overlay, staleParent}, detector)
	if tracker.Count() != 2 {
		t.Fatalf("tracked %d processes; want only the game root and child", tracker.Count())
	}
	tracker.Update([]processes.Info{gameChild, overlay, staleParent}, detector)
	if tracker.Count() != 1 {
		t.Fatalf("tracked %d processes after root exit; want only the game child", tracker.Count())
	}
	tracker.Update([]processes.Info{overlay, staleParent}, detector)
	if !tracker.Empty() {
		t.Fatal("non-game processes kept the session running")
	}
}

func TestDetectorRejectsKnownExecutableOutsideGameDirectory(t *testing.T) {
	detector := NewDetector(`C:\Games\Prey\Prey.exe`, "PreyGame.exe", `C:\Games\Prey`, nil, time.Unix(0, 100))
	outside := processes.Info{PID: 20, Name: "PreyGame.exe", Executable: `C:\Other\PreyGame.exe`, CreationTime: 200}
	if _, found := detector.Select([]processes.Info{outside}, false); found {
		t.Fatal("selected a process outside the game directory by name")
	}
	if detector.IsNewExactMatch(outside) {
		t.Fatal("tracked a process outside the game directory by name")
	}
}
