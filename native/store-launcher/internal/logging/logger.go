package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const maxLogSize int64 = 1024 * 1024

type Logger struct {
	mu       sync.Mutex
	path     string
	previous string
	file     *os.File
}

func New(requestedPath string) *Logger {
	logPath := requestedPath
	if logPath == "" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base, _ = os.UserConfigDir()
		}
		logPath = filepath.Join(base, "Steam ROM Manager", "logs", "store-launcher.log")
	}
	extension := filepath.Ext(logPath)
	previous := strings.TrimSuffix(logPath, extension) + ".previous" + extension
	logger := &Logger{path: logPath, previous: previous}
	logger.open()
	return logger
}

func (l *Logger) Printf(format string, arguments ...any) {
	if l == nil {
		return
	}
	line := fmt.Sprintf("%s ", time.Now().Format(time.RFC3339Nano)) + fmt.Sprintf(format, arguments...) + "\n"
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return
	}
	if info, err := l.file.Stat(); err == nil && info.Size()+int64(len(line)) > maxLogSize {
		l.rotate()
	}
	if l.file != nil {
		_, _ = l.file.WriteString(line)
	}
}

func (l *Logger) Close() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
}

func (l *Logger) open() {
	if l.path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err == nil {
		l.file = file
	}
}

func (l *Logger) rotate() {
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
	_ = os.Remove(l.previous)
	_ = os.Rename(l.path, l.previous)
	l.open()
}
