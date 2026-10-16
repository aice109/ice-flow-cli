// Package buildinfo holds application identity and build-time metadata.
//
// The variables are overridden at build time via -ldflags, e.g.
//
//	go build -ldflags "-X github.com/aice109/ice-flow-cli/internal/buildinfo.Version=1.2.3"
package buildinfo

import "runtime"

// Build-time variables. Do not set these in code; use -ldflags.
var (
	// Version is the semantic version of the build (e.g. "1.2.3").
	Version = "dev"
	// Commit is the short git commit hash the binary was built from.
	Commit = "none"
	// Date is the RFC3339 build timestamp.
	Date = "unknown"
)

// Application identity constants.
const (
	// AppName is the user-facing binary name.
	AppName = "iceflow"
	// BundleID is the reverse-DNS identifier used for launchd labels and
	// systemd unit names.
	BundleID = "com.aice109.iceflow"
	// RepoSlug is the GitHub "owner/repo" slug used by the self-updater.
	RepoSlug = "aice109/ice-flow-cli"
	// RepoURL is the project homepage.
	RepoURL = "https://github.com/aice109/ice-flow-cli"
	// Description is a one-line summary of the project.
	Description = "Local-first personal productivity CLI: task planning, reminders, and a self-built command dictionary with fuzzy search."
)

// Platform returns the current GOOS/GOARCH pair.
func Platform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}
