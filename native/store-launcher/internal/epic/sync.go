package epic

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Epic's LogCloudSync messages are diagnostic output, not a supported API.
// Keep this parser isolated so a launcher update can be handled in one place.
type SyncLog struct {
	Path      string
	file      os.FileInfo
	offset    int64
	Active    map[string]bool
	SuccessAt map[string]time.Time
	FailureAt map[string]time.Time
}

func DefaultLogPath() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "EpicGamesLauncher", "Saved", "Logs", "EpicGamesLauncher.log")
}

func NewSyncLog(path string) *SyncLog {
	if path == "" {
		path = DefaultLogPath()
	}
	return &SyncLog{Path: path, Active: map[string]bool{}, SuccessAt: map[string]time.Time{}, FailureAt: map[string]time.Time{}}
}

func (l *SyncLog) Poll() error {
	file, err := os.Open(l.Path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if l.file == nil || !os.SameFile(l.file, info) || info.Size() < l.offset {
		l.offset = 0
	}
	l.file = info
	if _, err := file.Seek(l.offset, io.SeekStart); err != nil {
		return err
	}
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil // Re-read an incomplete line on the next poll.
			}
			return err
		}
		l.offset += int64(len(line))
		l.consume(line)
	}
}

func (l *SyncLog) consume(line string) {
	if !strings.Contains(line, "LogCloudSync: Cloud Sync:") || len(line) < 25 || line[0] != '[' {
		return
	}
	closing := strings.IndexByte(line, ']')
	if closing < 0 {
		return
	}
	timestamp := line[1:closing]
	if len(timestamp) != len("2006.01.02-15.04.05:000") {
		return
	}
	timestamp = timestamp[:len(timestamp)-4] + "." + timestamp[len(timestamp)-3:]
	when, err := time.ParseInLocation("2006.01.02-15.04.05.000", timestamp, time.UTC)
	if err != nil {
		return
	}
	message := line[closing+1:]
	const started = "LogCloudSync: Cloud Sync: Sync Started for "
	const success = "LogCloudSync: Cloud Sync: Exiting Cloud Sync - SUCCESS - AppName: "
	const failure = "LogCloudSync: Cloud Sync: Exiting Cloud Sync - FAILURE - AppName: "
	switch {
	case strings.Contains(message, started):
		id := appID(strings.TrimSpace(message[strings.Index(message, started)+len(started):]))
		if id != "" {
			l.Active[id] = true
		}
	case strings.Contains(message, success):
		id := appID(strings.TrimSpace(message[strings.Index(message, success)+len(success):]))
		if id != "" {
			delete(l.Active, id)
			l.SuccessAt[id] = when
		}
	case strings.Contains(message, failure):
		id := appID(strings.TrimSpace(message[strings.Index(message, failure)+len(failure):]))
		if id != "" {
			delete(l.Active, id)
			l.FailureAt[id] = when
		}
	}
}

func appID(value string) string {
	parts := strings.Split(value, ":")
	return strings.ToLower(strings.TrimSpace(parts[len(parts)-1]))
}
