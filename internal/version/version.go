package version

import (
	"fmt"
	"runtime"
)

var (
	// Release is the semantic version (set via ldflags).
	Release = "dev"
	// GitCommit is the git commit hash (set via ldflags).
	GitCommit = "dev"
	// Implementation is the name of the implementation.
	Implementation = "cbt-api"
	// GOOS is the operating system.
	GOOS = runtime.GOOS
	// GOARCH is the architecture.
	GOARCH = runtime.GOARCH
)

// Full returns the full version string including implementation and version.
func Full() string {
	return fmt.Sprintf("%s/%s", Implementation, Short())
}

// Short returns the short version string.
func Short() string {
	return Release
}

// FullWithGOOS returns the full version with OS.
func FullWithGOOS() string {
	return fmt.Sprintf("%s/%s", Full(), GOOS)
}

// FullWithPlatform returns the full version with OS and architecture.
func FullWithPlatform() string {
	return fmt.Sprintf("%s/%s/%s", Full(), GOOS, GOARCH)
}
