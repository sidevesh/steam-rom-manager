package cli

import (
	"testing"

	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/storeuri"
)

func TestParseDefaultsExistingShortcutsToEpic(t *testing.T) {
	options, err := Parse([]string{
		"--uri", "com.epicgames.launcher://apps/Foo?action=launch",
		"--exe", `C:\Games\Foo\Foo.exe`,
	})
	if err != nil {
		t.Fatalf("Parse() returned an error: %v", err)
	}
	if options.Store != storeuri.Epic {
		t.Fatalf("Store = %q, want %q", options.Store, storeuri.Epic)
	}
}

func TestParseAcceptsEAAndUbisoft(t *testing.T) {
	tests := []struct {
		store string
		uri   string
		want  storeuri.Store
	}{
		{store: "ea", uri: "origin2://game/launch/?offerIds=123", want: storeuri.EA},
		{store: "ubisoft", uri: "uplay://launch/123", want: storeuri.Ubisoft},
	}

	for _, test := range tests {
		t.Run(test.store, func(t *testing.T) {
			options, err := Parse([]string{
				"--store", test.store,
				"--uri", test.uri,
				"--exe", `D:\Games\Foo\Foo.exe`,
			})
			if err != nil {
				t.Fatalf("Parse() returned an error: %v", err)
			}
			if options.Store != test.want {
				t.Fatalf("Store = %q, want %q", options.Store, test.want)
			}
		})
	}
}

func TestParseExecutableLaunchPreservesArguments(t *testing.T) {
	options, err := Parse([]string{
		"--store", "gog",
		"--launch-exe", `C:\Program Files\GOG Galaxy\GalaxyClient.exe`,
		"--launch-arg", "/command=runGame",
		"--launch-arg", "/gameId=123 456",
		"--launch-cwd", `C:\Program Files\GOG Galaxy`,
		"--exe", `D:\Games\Foo\Foo.exe`,
	})
	if err != nil {
		t.Fatalf("Parse() returned an error: %v", err)
	}
	if len(options.LaunchArguments) != 2 || options.LaunchArguments[1] != "/gameId=123 456" {
		t.Fatalf("LaunchArguments = %#v", options.LaunchArguments)
	}
}

func TestParseExecutableLaunchAllowsInstallDirectoryOnly(t *testing.T) {
	_, err := Parse([]string{
		"--store", "amazon",
		"--launch-exe", `C:\Amazon Games\Amazon Games.exe`,
		"--launch-arg", "amazon-games://play/123",
		"--install-dir", `D:\Games\Foo`,
	})
	if err != nil {
		t.Fatalf("Parse() returned an error: %v", err)
	}
}

func TestParseRejectsMixedLaunchModes(t *testing.T) {
	_, err := Parse([]string{
		"--uri", "com.epicgames.launcher://apps/Foo?action=launch",
		"--launch-exe", `C:\Store\Store.exe`,
		"--exe", `D:\Games\Foo.exe`,
	})
	if err == nil {
		t.Fatal("Parse() accepted both URI and executable launch")
	}
}

func TestParseAcceptsXboxAppIdentity(t *testing.T) {
	options, err := Parse([]string{
		"--store", "xbox",
		"--aumid", "Example.Game_123abc!Game",
		"--exe", `D:\XboxGames\Game\Content\GameLaunchHelper.exe`,
		"--install-dir", `D:\XboxGames\Game\Content`,
	})
	if err != nil {
		t.Fatalf("Parse() returned an error: %v", err)
	}
	if options.AppUserModelID != "Example.Game_123abc!Game" {
		t.Fatalf("AppUserModelID = %q", options.AppUserModelID)
	}
}

func TestParseRejectsInvalidXboxAppIdentity(t *testing.T) {
	for _, id := range []string{"shell:AppsFolder\\Example.Game!Game", "Example.Game", "Example.Game!", "Example.Game!Game!Other"} {
		_, err := Parse([]string{
			"--store", "xbox", "--aumid", id,
			"--exe", `D:\XboxGames\Game\GameLaunchHelper.exe`,
		})
		if err == nil {
			t.Errorf("Parse() accepted invalid AppUserModelID %q", id)
		}
	}
}
