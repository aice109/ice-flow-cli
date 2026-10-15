// Package service installs and manages a periodic background job that runs
// `iflow remind daemon --tick`, so reminders fire even when the CLI is not
// running.
//
// On macOS it writes a launchd agent; on Linux it prefers a systemd --user
// timer and falls back to a crontab entry when systemd is unavailable.
package service

import "github.com/aice109/ice-flow-cli/internal/buildinfo"

// Label is the reverse-DNS identifier used for the launchd label.
func Label() string { return buildinfo.BundleID }

// Install registers the periodic job for the executable at exe.
// It returns a human-readable description of what was installed.
func Install(exe string) (string, error) { return install(exe) }

// Uninstall removes the periodic job.
func Uninstall() (string, error) { return uninstall() }

// Status reports whether the periodic job is installed and running.
func Status() (string, error) { return status() }
