package cli

import (
	"flag"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/epic"
)

type Options struct {
	URI                string
	ExpectedExecutable string
	InstallDirectory   string
	ProcessOverride    string
	StartTimeout       time.Duration
	PollInterval       time.Duration
	HandoffGrace       time.Duration
	LogFile            string
	AllowExistingMatch bool
}

func Parse(args []string) (Options, error) {
	options := Options{
		StartTimeout: 60 * time.Second,
		PollInterval: 500 * time.Millisecond,
		HandoffGrace: 5 * time.Second,
	}

	flags := flag.NewFlagSet("srm-store-launcher", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&options.URI, "uri", "", "Epic launcher URI")
	flags.StringVar(&options.ExpectedExecutable, "exe", "", "expected executable path")
	flags.StringVar(&options.InstallDirectory, "install-dir", "", "game installation directory")
	flags.StringVar(&options.ProcessOverride, "process", "", "expected long-running process name")
	flags.DurationVar(&options.StartTimeout, "start-timeout", options.StartTimeout, "process discovery timeout")
	flags.DurationVar(&options.PollInterval, "poll-interval", options.PollInterval, "process polling interval")
	flags.DurationVar(&options.HandoffGrace, "handoff-grace", options.HandoffGrace, "process handoff grace period")
	flags.StringVar(&options.LogFile, "log-file", "", "diagnostic log path")
	flags.BoolVar(&options.AllowExistingMatch, "allow-existing-match", false, "allow attaching to a matching process that already exists")
	if err := flags.Parse(args); err != nil {
		return options, err
	}
	if flags.NArg() != 0 {
		return options, fmt.Errorf("unexpected positional arguments")
	}
	if err := epic.ValidateURI(options.URI); err != nil {
		return options, err
	}
	if !isAbsoluteWindowsPath(options.ExpectedExecutable) {
		return options, fmt.Errorf("--exe must be an absolute Windows path")
	}
	if options.InstallDirectory != "" && !isAbsoluteWindowsPath(options.InstallDirectory) {
		return options, fmt.Errorf("--install-dir must be an absolute Windows path")
	}
	if options.ProcessOverride != "" && windowsBase(options.ProcessOverride) != options.ProcessOverride {
		return options, fmt.Errorf("--process must be a process name, not a path")
	}
	if options.StartTimeout <= 0 || options.PollInterval <= 0 || options.HandoffGrace <= 0 {
		return options, fmt.Errorf("timeouts, polling interval, and handoff grace must be positive")
	}
	return options, nil
}

func (o Options) ExpectedProcessName() string {
	if o.ProcessOverride != "" {
		return o.ProcessOverride
	}
	return windowsBase(o.ExpectedExecutable)
}

func isAbsoluteWindowsPath(value string) bool {
	if len(value) >= 3 && isLetter(value[0]) && value[1] == ':' && isSeparator(value[2]) {
		return true
	}
	if len(value) < 5 || !isSeparator(value[0]) || !isSeparator(value[1]) {
		return false
	}
	parts := strings.Split(strings.Trim(strings.ReplaceAll(value[2:], "\\", "/"), "/"), "/")
	return len(parts) >= 2 && parts[0] != "" && parts[1] != ""
}

func windowsBase(value string) string {
	return path.Base(strings.ReplaceAll(value, "\\", "/"))
}

func isLetter(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func isSeparator(value byte) bool {
	return value == '\\' || value == '/'
}
