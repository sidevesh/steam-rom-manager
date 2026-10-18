//go:build windows

package recovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/cli"
	processes "github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/process"
)

const heartbeatMaxAge = 10 * time.Second

type heartbeat struct {
	Agent processes.Info `json:"agent"`
}

// Record survives an abrupt helper termination. The agent checks process
// creation times before acting, so a reused PID never identifies a new game.
type Record struct {
	Helper       processes.Info `json:"helper"`
	Game         processes.Info `json:"game"`
	Options      cli.Options    `json:"options"`
	RegisteredAt time.Time      `json:"registeredAt"`
}

func Directory() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "Steam ROM Manager", "agent")
}

func heartbeatPath() string { return filepath.Join(Directory(), "heartbeat") }

func SessionPath(helper processes.Info) string {
	return filepath.Join(Directory(), fmt.Sprintf("session-%d-%d.json", helper.PID, helper.CreationTime))
}

func AgentReady() error {
	info, err := os.Stat(heartbeatPath())
	if err != nil {
		return fmt.Errorf("store agent is not running: %w", err)
	}
	if time.Since(info.ModTime()) > heartbeatMaxAge {
		return fmt.Errorf("store agent heartbeat is stale (%s)", time.Since(info.ModTime()).Round(time.Second))
	}
	data, err := os.ReadFile(heartbeatPath())
	if err != nil {
		return err
	}
	var state heartbeat
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	snapshot, err := (processes.WindowsSource{}).Snapshot()
	if err != nil {
		return err
	}
	for _, current := range snapshot {
		if current.Identity() == state.Agent.Identity() && strings.EqualFold(current.Name, state.Agent.Name) && current.CreationTime != 0 {
			return nil
		}
	}
	return errors.New("store agent heartbeat process is no longer running")
}

func WriteHeartbeat() error {
	if err := os.MkdirAll(Directory(), 0700); err != nil {
		return err
	}
	snapshot, err := (processes.WindowsSource{}).Snapshot()
	if err != nil {
		return err
	}
	for _, current := range snapshot {
		if current.PID == uint32(os.Getpid()) && current.CreationTime != 0 {
			data, err := json.Marshal(heartbeat{Agent: current})
			if err != nil {
				return err
			}
			temporary := heartbeatPath() + ".tmp"
			if err := os.WriteFile(temporary, data, 0600); err != nil {
				return err
			}
			return os.Rename(temporary, heartbeatPath())
		}
	}
	return errors.New("store agent process identity unavailable")
}

func Register(record Record) (string, error) {
	if err := AgentReady(); err != nil {
		return "", err
	}
	if record.Helper.CreationTime == 0 || record.Game.CreationTime == 0 {
		return "", errors.New("process identity lacks creation time")
	}
	if err := os.MkdirAll(Directory(), 0700); err != nil {
		return "", err
	}
	path := SessionPath(record.Helper)
	data, err := json.Marshal(record)
	if err != nil {
		return "", err
	}
	temporary := fmt.Sprintf("%s.tmp-%d", path, os.Getpid())
	if err := os.WriteFile(temporary, data, 0600); err != nil {
		return "", err
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return "", err
	}
	return path, nil
}

func Complete(path string) error {
	if path == "" {
		return nil
	}
	return os.Remove(path)
}

func Sessions() ([]string, error) {
	entries, err := os.ReadDir(Directory())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".json") || strings.HasSuffix(entry.Name(), ".recovering")) {
			paths = append(paths, filepath.Join(Directory(), entry.Name()))
		}
	}
	return paths, nil
}

func Read(path string) (Record, error) {
	var record Record
	data, err := os.ReadFile(path)
	if err != nil {
		return record, err
	}
	err = json.Unmarshal(data, &record)
	return record, err
}
