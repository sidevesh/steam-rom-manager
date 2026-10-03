package cli

import (
	"flag"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/storeuri"
)

type Options struct {
	Store              storeuri.Store
	URI                string
	LaunchExecutable   string
	LaunchArguments    []string
	LaunchDirectory    string
	AppUserModelID     string
	ExpectedExecutable string
	InstallDirectory   string
	ProcessOverride    string
	StartTimeout       time.Duration
	PollInterval       time.Duration
	HandoffGrace       time.Duration
	LogFile            string
	AllowExistingMatch bool
}

type argumentList []string

func (a *argumentList) String() string { return fmt.Sprint([]string(*a)) }
func (a *argumentList) Set(value string) error {
	*a = append(*a, value)
	return nil
}

func Parse(args []string) (Options, error) {
	options := Options{
		Store:        storeuri.Epic,
		StartTimeout: 60 * time.Second,
		PollInterval: 500 * time.Millisecond,
		HandoffGrace: 5 * time.Second,
	}

	flags := flag.NewFlagSet("srm-store-launcher", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	storeName := string(options.Store)
	flags.StringVar(&storeName, "store", storeName, "store profile")
	flags.StringVar(&options.URI, "uri", "", "store launcher URI")
	flags.StringVar(&options.LaunchExecutable, "launch-exe", "", "store launcher executable")
	flags.Var((*argumentList)(&options.LaunchArguments), "launch-arg", "one launcher argument; may be repeated")
	flags.StringVar(&options.LaunchDirectory, "launch-cwd", "", "store launcher working directory")
	flags.StringVar(&options.AppUserModelID, "aumid", "", "Windows app user model ID to activate")
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
	store, err := storeuri.ParseStore(storeName)
	if err != nil {
		return options, err
	}
	options.Store = store
	launchModes := 0
	for _, present := range []bool{options.URI != "", options.LaunchExecutable != "", options.AppUserModelID != ""} {
		if present {
			launchModes++
		}
	}
	if launchModes != 1 {
		return options, fmt.Errorf("exactly one of --uri, --launch-exe, and --aumid is required")
	}
	if options.URI != "" {
		if err := storeuri.Validate(options.Store, options.URI); err != nil {
			return options, err
		}
		if options.LaunchDirectory != "" || len(options.LaunchArguments) != 0 {
			return options, fmt.Errorf("--launch-arg and --launch-cwd require --launch-exe")
		}
	} else if options.LaunchExecutable != "" && !isAbsoluteWindowsPath(options.LaunchExecutable) {
		return options, fmt.Errorf("--launch-exe must be an absolute Windows path")
	}
	if options.AppUserModelID != "" {
		if options.Store != storeuri.Xbox {
			return options, fmt.Errorf("--aumid requires --store xbox")
		}
		if !validAppUserModelID(options.AppUserModelID) {
			return options, fmt.Errorf("--aumid must be a package family and application ID separated by !")
		}
		if options.LaunchDirectory != "" || len(options.LaunchArguments) != 0 {
			return options, fmt.Errorf("--launch-arg and --launch-cwd require --launch-exe")
		}
	}
	if options.LaunchDirectory != "" && !isAbsoluteWindowsPath(options.LaunchDirectory) {
		return options, fmt.Errorf("--launch-cwd must be an absolute Windows path")
	}
	if options.ExpectedExecutable != "" && !isAbsoluteWindowsPath(options.ExpectedExecutable) {
		return options, fmt.Errorf("--exe must be an absolute Windows path")
	}
	if options.ExpectedExecutable == "" && (options.URI != "" || options.InstallDirectory == "") {
		return options, fmt.Errorf("--exe is required unless --launch-exe or --aumid has --install-dir")
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

func validAppUserModelID(value string) bool {
	if strings.Count(value, "!") != 1 || strings.TrimSpace(value) != value {
		return false
	}
	parts := strings.SplitN(value, "!", 2)
	return parts[0] != "" && parts[1] != "" && !strings.ContainsAny(value, "\\/:\x00\r\n\t ")
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
