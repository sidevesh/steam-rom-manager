package epic

import (
	"fmt"
	"net/url"
	"strings"
)

func ValidateURI(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("parse Epic URI: %w", err)
	}
	if !parsed.IsAbs() || !strings.EqualFold(parsed.Scheme, "com.epicgames.launcher") {
		return fmt.Errorf("URI must use the com.epicgames.launcher scheme")
	}
	if !strings.EqualFold(parsed.Host, "apps") || strings.Trim(parsed.Path, "/") == "" {
		return fmt.Errorf("URI must target the apps endpoint and include an app name")
	}
	if !strings.EqualFold(parsed.Query().Get("action"), "launch") {
		return fmt.Errorf("URI query must contain action=launch")
	}
	return nil
}

// SanitizedURI retains the endpoint and app identifier while dropping query
// values that may be added by Epic in the future.
func SanitizedURI(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return "<invalid>"
	}
	return parsed.Scheme + "://" + parsed.Host + parsed.EscapedPath()
}
