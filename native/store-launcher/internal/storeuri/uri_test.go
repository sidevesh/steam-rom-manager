package storeuri

import "testing"

func TestValidateSupportedStoreURIs(t *testing.T) {
	tests := []struct {
		name  string
		store Store
		uri   string
	}{
		{
			name:  "Epic",
			store: Epic,
			uri:   "com.epicgames.launcher://apps/Foo?action=launch&silent=true",
		},
		{
			name:  "EA",
			store: EA,
			uri:   "origin2://game/launch/?offerIds=OFB-EAST:12345",
		},
		{
			name:  "Ubisoft",
			store: Ubisoft,
			uri:   "uplay://launch/1234",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := Validate(test.store, test.uri); err != nil {
				t.Fatalf("Validate() returned an error: %v", err)
			}
		})
	}
}

func TestValidateRejectsMismatchedOrIncompleteURIs(t *testing.T) {
	tests := []struct {
		name  string
		store Store
		uri   string
	}{
		{name: "Epic with EA URI", store: Epic, uri: "origin2://game/launch/?offerIds=123"},
		{name: "Epic without action", store: Epic, uri: "com.epicgames.launcher://apps/Foo"},
		{name: "EA without offer", store: EA, uri: "origin2://game/launch/"},
		{name: "Ubisoft without app ID", store: Ubisoft, uri: "uplay://launch/"},
		{name: "arbitrary scheme", store: Ubisoft, uri: "file:///C:/Windows/System32/calc.exe"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := Validate(test.store, test.uri); err == nil {
				t.Fatal("Validate() unexpectedly accepted URI")
			}
		})
	}
}

func TestParseStoreIsCaseInsensitive(t *testing.T) {
	store, err := ParseStore(" Ubisoft ")
	if err != nil {
		t.Fatalf("ParseStore() returned an error: %v", err)
	}
	if store != Ubisoft {
		t.Fatalf("ParseStore() = %q, want %q", store, Ubisoft)
	}
}

func TestSanitizedDropsQuery(t *testing.T) {
	got := Sanitized("origin2://game/launch/?offerIds=secret")
	want := "origin2://game/launch/"
	if got != want {
		t.Fatalf("Sanitized() = %q, want %q", got, want)
	}
}
