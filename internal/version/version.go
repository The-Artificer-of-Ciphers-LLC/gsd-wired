package version

import (
	"encoding/json"
	"fmt"
	"runtime"
	"runtime/debug"
)

// Version is the fallback semver string used when no build info is available.
const Version = "0.1.0"

// Package-level vars injected by goreleaser ldflags at release time.
// These take precedence over ReadBuildInfo when non-empty.
var (
	version string // set by goreleaser ldflags: -X ...version.version={{.Version}}
	commit  string // set by goreleaser ldflags: -X ...version.commit={{.Commit}}
	date    string // set by goreleaser ldflags: -X ...version.date={{.Date}}
)

// BuildInfo holds structured version information.
type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
}

// String returns the version in "VERSION (COMMIT)" format per D-12.
func (b BuildInfo) String() string {
	c := b.Commit
	if c == "" {
		c = "unknown"
	}
	return fmt.Sprintf("%s (%s)", b.Version, c)
}

// JSON returns indented JSON representation of the BuildInfo.
func (b BuildInfo) JSON() (string, error) {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetInfo returns BuildInfo populated from ldflags (if set at link time) or
// from runtime/debug.ReadBuildInfo as a fallback (for `go install` users).
func GetInfo() BuildInfo {
	bi := BuildInfo{
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}

	// If goreleaser injected ldflags, use them (release builds).
	if version != "" {
		bi.Version = version
		bi.Commit = commit
		bi.Date = date
		return bi
	}

	// Fallback: read vcs info embedded by go build / go install.
	bi.Version = Version
	info, ok := debug.ReadBuildInfo()
	if !ok {
		bi.Commit = "unknown"
		return bi
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			if len(s.Value) >= 7 {
				bi.Commit = s.Value[:7]
			} else if len(s.Value) > 0 {
				bi.Commit = s.Value
			} else {
				bi.Commit = "unknown"
			}
			break
		}
	}
	if bi.Commit == "" {
		bi.Commit = "unknown"
	}
	return bi
}

// String returns the version string in "SEMVER (HASH)" format.
// Delegates to GetInfo().String() for backward compatibility.
func String() string {
	return GetInfo().String()
}
