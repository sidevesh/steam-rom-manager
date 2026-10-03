package storeuri

import (
	"fmt"
	"net/url"
	"strings"
)

type Store string

const (
	Epic    Store = "epic"
	EA      Store = "ea"
	Ubisoft Store = "ubisoft"
	Amazon  Store = "amazon"
	GOG     Store = "gog"
	Xbox    Store = "xbox"
)

func ParseStore(value string) (Store, error) {
	store := Store(strings.ToLower(strings.TrimSpace(value)))
	switch store {
	case Epic, EA, Ubisoft, Amazon, GOG, Xbox:
		return store, nil
	default:
		return "", fmt.Errorf("unsupported store %q", value)
	}
}

func Validate(store Store, value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("parse %s URI: %w", store, err)
	}
	if !parsed.IsAbs() {
		return fmt.Errorf("%s URI must be absolute", store)
	}

	switch store {
	case Epic:
		if !strings.EqualFold(parsed.Scheme, "com.epicgames.launcher") {
			return fmt.Errorf("Epic URI must use the com.epicgames.launcher scheme")
		}
		if !strings.EqualFold(parsed.Host, "apps") || strings.Trim(parsed.Path, "/") == "" {
			return fmt.Errorf("Epic URI must target the apps endpoint and include an app name")
		}
		if !strings.EqualFold(parsed.Query().Get("action"), "launch") {
			return fmt.Errorf("Epic URI query must contain action=launch")
		}
	case EA:
		if !strings.EqualFold(parsed.Scheme, "origin2") {
			return fmt.Errorf("EA URI must use the origin2 scheme")
		}
		if !strings.EqualFold(parsed.Host, "game") || !strings.EqualFold(strings.Trim(parsed.Path, "/"), "launch") {
			return fmt.Errorf("EA URI must target the game/launch endpoint")
		}
		if strings.TrimSpace(parsed.Query().Get("offerIds")) == "" {
			return fmt.Errorf("EA URI query must contain offerIds")
		}
	case Ubisoft:
		if !strings.EqualFold(parsed.Scheme, "uplay") {
			return fmt.Errorf("Ubisoft URI must use the uplay scheme")
		}
		if !strings.EqualFold(parsed.Host, "launch") || strings.Trim(parsed.Path, "/") == "" {
			return fmt.Errorf("Ubisoft URI must target the launch endpoint and include an app ID")
		}
	default:
		return fmt.Errorf("unsupported store %q", store)
	}

	return nil
}

// Sanitized retains the protocol endpoint and path while dropping query
// values that could contain identifiers or tokens added by a store later.
func Sanitized(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return "<invalid>"
	}
	return parsed.Scheme + "://" + parsed.Host + parsed.EscapedPath()
}
