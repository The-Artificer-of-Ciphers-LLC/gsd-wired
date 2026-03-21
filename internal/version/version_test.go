package version

import (
	"regexp"
	"testing"
)

func TestStringFormat(t *testing.T) {
	result := String()
	matched, err := regexp.MatchString(`^\d+\.\d+\.\d+ \(.+\)$`, result)
	if err != nil {
		t.Fatalf("regexp error: %v", err)
	}
	if !matched {
		t.Errorf("version.String() = %q, does not match expected format ^\\d+\\.\\d+\\.\\d+ \\(.+\\)$", result)
	}
}

func TestStringContainsFallback(t *testing.T) {
	result := String()
	if len(result) < 5 || result[:5] != "0.1.0" {
		t.Errorf("version.String() = %q, expected to start with '0.1.0'", result)
	}
}
