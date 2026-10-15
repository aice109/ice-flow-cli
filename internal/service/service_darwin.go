//go:build darwin

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/aice109/ice-flow-cli/internal/config"
	"github.com/aice109/ice-flow-cli/internal/i18n"
)

func plistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", Label()+".plist")
}

func domainTarget() string  { return fmt.Sprintf("gui/%d", os.Getuid()) }
func serviceTarget() string { return domainTarget() + "/" + Label() }

func plistBody(exe string) string {
	logPath := config.StatePath("daemon.log")
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>%s</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
    <string>remind</string>
    <string>daemon</string>
    <string>--tick</string>
  </array>
  <key>StartInterval</key>
  <integer>60</integer>
  <key>RunAtLoad</key>
  <false/>
  <key>StandardOutPath</key>
  <string>%s</string>
  <key>StandardErrorPath</key>
  <string>%s</string>
</dict>
</plist>
`, Label(), exe, logPath, logPath)
}

func install(exe string) (string, error) {
	if err := os.MkdirAll(config.DataDir(), 0o755); err != nil {
		return "", err
	}
	p := plistPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(p, []byte(plistBody(exe)), 0o644); err != nil {
		return "", err
	}
	// Reload: bootout is a no-op when not loaded.
	_ = exec.Command("launchctl", "bootout", serviceTarget()).Run()
	if out, err := exec.Command("launchctl", "bootstrap", domainTarget(), p).CombinedOutput(); err != nil {
		if out2, err2 := exec.Command("launchctl", "load", "-w", p).CombinedOutput(); err2 != nil {
			return "", fmt.Errorf("launchctl bootstrap: %v (%s); launchctl load: %v (%s)", err, out, err2, out2)
		}
	}
	_ = exec.Command("launchctl", "enable", serviceTarget()).Run()
	return i18n.Tf("installed launchd agent %s\n  plist:  %s\n  action: %s remind daemon --tick (every 60s)", Label(), p, exe), nil
}

func uninstall() (string, error) {
	p := plistPath()
	_ = exec.Command("launchctl", "bootout", serviceTarget()).Run()
	_ = exec.Command("launchctl", "unload", "-w", p).Run()
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return "", err
	}
	return i18n.Tf("removed launchd agent %s", Label()), nil
}

func status() (string, error) {
	p := plistPath()
	if _, err := os.Stat(p); err != nil {
		return i18n.Tf("not installed\n  (no plist at %s)", p), nil
	}
	if _, err := exec.Command("launchctl", "print", serviceTarget()).CombinedOutput(); err != nil {
		return i18n.Tf("installed but not loaded\n  plist: %s", p), nil
	}
	return i18n.Tf("installed and loaded\n  plist: %s", p), nil
}
