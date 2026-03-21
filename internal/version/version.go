package version

import (
	"fmt"
	"runtime/debug"
)

// Version is the fallback semver string used when no build info is available.
const Version = "0.1.0"

// String returns the version string in "SEMVER (HASH)" format.
// It reads vcs.revision from runtime/debug build info embedded by go build.
// If the revision is not available, it returns "0.1.0 (unknown)".
func String() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return fmt.Sprintf("%s (unknown)", Version)
	}
	hash := "unknown"
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			if len(s.Value) >= 7 {
				hash = s.Value[:7]
			} else if len(s.Value) > 0 {
				hash = s.Value
			}
			break
		}
	}
	return fmt.Sprintf("%s (%s)", Version, hash)
}
