package session

import (
	"fmt"

	processes "github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/process"
)

type Tracker struct {
	tracked map[processes.Identity]processes.Info
}

func NewTracker(root processes.Info) *Tracker {
	return &Tracker{tracked: map[processes.Identity]processes.Info{root.Identity(): root}}
}

func (t *Tracker) Empty() bool {
	return len(t.tracked) == 0
}

func (t *Tracker) Count() int {
	return len(t.tracked)
}

func (t *Tracker) Add(process processes.Info) bool {
	identity := process.Identity()
	if _, exists := t.tracked[identity]; exists {
		return false
	}
	t.tracked[identity] = process
	return true
}

func (t *Tracker) Update(snapshot []processes.Info, detector *Detector) []string {
	events := make([]string, 0)
	current := make(map[processes.Identity]processes.Info, len(snapshot))
	byPID := make(map[uint32]processes.Info, len(snapshot))
	for _, process := range snapshot {
		current[process.Identity()] = process
		byPID[process.PID] = process
	}

	for identity, tracked := range t.tracked {
		if _, alive := current[identity]; alive {
			continue
		}
		delete(t.tracked, identity)
		if replacement, reused := byPID[identity.PID]; reused && replacement.CreationTime != identity.CreationTime {
			events = append(events, fmt.Sprintf("removed pid=%d name=%q (PID reused)", tracked.PID, tracked.Name))
		} else {
			events = append(events, fmt.Sprintf("removed pid=%d name=%q (exited)", tracked.PID, tracked.Name))
		}
	}

	for {
		trackedPIDs := make(map[uint32]bool, len(t.tracked))
		for _, tracked := range t.tracked {
			trackedPIDs[tracked.PID] = true
		}
		added := false
		for _, process := range snapshot {
			if trackedPIDs[process.ParentPID] && t.Add(process) {
				events = append(events, fmt.Sprintf("added descendant pid=%d parent=%d name=%q", process.PID, process.ParentPID, process.Name))
				added = true
			}
		}
		if !added {
			break
		}
	}

	for _, process := range snapshot {
		if detector.IsNewExactMatch(process) && t.Add(process) {
			events = append(events, fmt.Sprintf("added independent match pid=%d name=%q", process.PID, process.Name))
		}
	}
	return events
}
