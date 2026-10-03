package session

import (
	"path"
	"sort"
	"strings"
	"time"

	processes "github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/process"
)

const creationTimeTolerance = 3 * time.Second

type Match struct {
	Process processes.Info
	Reason  string
}

type Detector struct {
	expectedExecutable string
	expectedName       string
	processOverride    string
	installDirectory   string
	baseline           map[uint32]uint64
	launchTime         uint64
	fallbackSeen       map[processes.Identity]int
}

func NewDetector(expectedExecutable, processOverride, installDirectory string, baseline []processes.Info, launchTime time.Time) *Detector {
	baselinePIDs := make(map[uint32]uint64, len(baseline))
	for _, process := range baseline {
		baselinePIDs[process.PID] = process.CreationTime
	}
	return &Detector{
		expectedExecutable: normalizeWindowsPath(expectedExecutable),
		expectedName:       strings.ToLower(windowsBase(expectedExecutable)),
		processOverride:    strings.ToLower(processOverride),
		installDirectory:   normalizeWindowsPath(installDirectory),
		baseline:           baselinePIDs,
		launchTime:         uint64(launchTime.UnixNano()),
		fallbackSeen:       make(map[processes.Identity]int),
	}
}

func (d *Detector) Select(snapshot []processes.Info, allowExisting bool) (Match, bool) {
	eligible := make([]processes.Info, 0, len(snapshot))
	current := make(map[processes.Identity]bool, len(snapshot))
	for _, process := range snapshot {
		if d.isEligible(process, allowExisting) {
			eligible = append(eligible, process)
			current[process.Identity()] = true
		}
	}
	sort.Slice(eligible, func(i, j int) bool {
		_, iExisted := d.baseline[eligible[i].PID]
		_, jExisted := d.baseline[eligible[j].PID]
		if iExisted != jExisted {
			return !iExisted
		}
		if eligible[i].CreationTime != eligible[j].CreationTime {
			return eligible[i].CreationTime < eligible[j].CreationTime
		}
		return eligible[i].PID < eligible[j].PID
	})

	for _, process := range eligible {
		if d.matchesExecutable(process) {
			return Match{Process: process, Reason: "exact executable path"}, true
		}
	}
	if d.processOverride != "" {
		for _, process := range eligible {
			if strings.EqualFold(process.Name, d.processOverride) {
				return Match{Process: process, Reason: "process override"}, true
			}
		}
	}
	for _, process := range eligible {
		if strings.EqualFold(process.Name, d.expectedName) {
			return Match{Process: process, Reason: "expected process name"}, true
		}
	}

	for identity := range d.fallbackSeen {
		if !current[identity] {
			delete(d.fallbackSeen, identity)
		}
	}
	for _, process := range eligible {
		if d.isUnderInstallDirectory(process.Executable) {
			identity := process.Identity()
			d.fallbackSeen[identity]++
			if d.fallbackSeen[identity] >= 2 {
				return Match{Process: process, Reason: "stable process beneath install directory"}, true
			}
		}
	}
	return Match{}, false
}

func (d *Detector) IsNewExactMatch(process processes.Info) bool {
	if !d.isEligible(process, false) {
		return false
	}
	return d.matchesExecutable(process) ||
		d.processOverride != "" && strings.EqualFold(process.Name, d.processOverride) ||
		strings.EqualFold(process.Name, d.expectedName)
}

func (d *Detector) TimeoutCandidates(snapshot []processes.Info, limit int) []processes.Info {
	candidates := make([]processes.Info, 0, len(snapshot))
	for _, process := range snapshot {
		if d.isEligible(process, false) {
			candidates = append(candidates, process)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		iUnder := d.isUnderInstallDirectory(candidates[i].Executable)
		jUnder := d.isUnderInstallDirectory(candidates[j].Executable)
		if iUnder != jUnder {
			return iUnder
		}
		iDistance := editDistance(strings.ToLower(candidates[i].Name), d.expectedName)
		jDistance := editDistance(strings.ToLower(candidates[j].Name), d.expectedName)
		if iDistance != jDistance {
			return iDistance < jDistance
		}
		if candidates[i].CreationTime != candidates[j].CreationTime {
			return candidates[i].CreationTime > candidates[j].CreationTime
		}
		return candidates[i].PID < candidates[j].PID
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

func (d *Detector) isEligible(process processes.Info, allowExisting bool) bool {
	if process.CreationTime == 0 {
		return false
	}
	if _, existed := d.baseline[process.PID]; existed {
		return allowExisting
	}
	tolerance := uint64(creationTimeTolerance.Nanoseconds())
	return process.CreationTime+tolerance >= d.launchTime
}

func (d *Detector) matchesExecutable(process processes.Info) bool {
	return process.Executable != "" && normalizeWindowsPath(process.Executable) == d.expectedExecutable
}

func (d *Detector) isUnderInstallDirectory(executable string) bool {
	if d.installDirectory == "" || executable == "" {
		return false
	}
	normalized := normalizeWindowsPath(executable)
	return normalized == d.installDirectory || strings.HasPrefix(normalized, d.installDirectory+"/")
}

func normalizeWindowsPath(value string) string {
	if value == "" {
		return ""
	}
	normalized := strings.ReplaceAll(value, "\\", "/")
	normalized = path.Clean(normalized)
	return strings.ToLower(strings.TrimSuffix(normalized, "/"))
}

func windowsBase(value string) string {
	return path.Base(strings.ReplaceAll(value, "\\", "/"))
}

func editDistance(left, right string) int {
	previous := make([]int, len(right)+1)
	for index := range previous {
		previous[index] = index
	}
	for i := 1; i <= len(left); i++ {
		current := make([]int, len(right)+1)
		current[0] = i
		for j := 1; j <= len(right); j++ {
			cost := 0
			if left[i-1] != right[j-1] {
				cost = 1
			}
			current[j] = min(current[j-1]+1, previous[j]+1, previous[j-1]+cost)
		}
		previous = current
	}
	return previous[len(right)]
}
