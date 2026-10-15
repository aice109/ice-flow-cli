// Package notify delivers reminder notifications using the best backend
// available on the host.
package notify

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Backend names accepted in config.
const (
	Auto       = "auto"
	OSAScript  = "osascript"
	NotifySend = "notify-send"
	Stdout     = "stdout"
	Off        = "off"
)

// Notifier sends a notification.
type Notifier struct {
	Backend string // one of the Backend constants
	Sound   bool
}

// New builds a Notifier from the configured backend name.
func New(backend string, sound bool) Notifier {
	return Notifier{Backend: strings.ToLower(strings.TrimSpace(backend)), Sound: sound}
}

// Send delivers a notification. A delivery error is returned but callers may
// treat it as non-fatal; stdout is always a safe fallback.
func (n Notifier) Send(title, body string) error {
	switch n.resolve() {
	case Off:
		return nil
	case OSAScript:
		return n.osascript(title, body)
	case NotifySend:
		return n.notifySend(title, body)
	default:
		return n.stdout(title, body)
	}
}

func (n Notifier) resolve() string {
	switch n.Backend {
	case OSAScript, NotifySend, Stdout, Off:
		return n.Backend
	default:
		// auto-detect
		if runtime.GOOS == "darwin" {
			if _, err := exec.LookPath("osascript"); err == nil {
				return OSAScript
			}
		}
		if _, err := exec.LookPath("notify-send"); err == nil {
			return NotifySend
		}
		return Stdout
	}
}

func (n Notifier) osascript(title, body string) error {
	script := fmt.Sprintf(`display notification %s with title %s`, appleStr(body), appleStr(title))
	if n.Sound {
		script = fmt.Sprintf(`display notification %s with title %s sound name "Ping"`, appleStr(body), appleStr(title))
	}
	return exec.Command("osascript", "-e", script).Run()
}

func (n Notifier) notifySend(title, body string) error {
	args := []string{title, body}
	return exec.Command("notify-send", args...).Run()
}

func (n Notifier) stdout(title, body string) error {
	fmt.Fprintf(os.Stderr, "\a\n🔔 %s\n   %s\n", title, body)
	return nil
}

// appleStr quotes a string for AppleScript, escaping backslashes and quotes.
func appleStr(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	// Strip newlines which break the single-line AppleScript statement.
	s = strings.ReplaceAll(s, "\n", " ")
	return `"` + s + `"`
}
