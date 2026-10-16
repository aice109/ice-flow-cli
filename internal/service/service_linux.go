//go:build linux

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aice109/ice-flow-cli/internal/buildinfo"
	"github.com/aice109/ice-flow-cli/internal/i18n"
)

const (
	unitName   = "iceflow"
	cronMarker = "# ice-flow (" + buildinfo.BundleID + ")"
)

func unitDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user")
}

func servicePath() string { return filepath.Join(unitDir(), unitName+".service") }

// timerPath is the legacy systemd timer installed by versions before the
// resident scheduler. It is removed on install and uninstall so an upgraded
// system does not run both a timer and the service.
func timerPath() string { return filepath.Join(unitDir(), unitName+".timer") }

func serviceBody(exe string) string {
	return fmt.Sprintf(`[Unit]
Description=ice-flow reminder scheduler (%s)

[Service]
Type=simple
ExecStart=%s remind daemon
Restart=always
RestartSec=1

[Install]
WantedBy=default.target
`, Label(), exe)
}

func hasSystemd() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

func install(exe string) (string, error) {
	if hasSystemd() {
		return installSystemd(exe)
	}
	return installCron(exe)
}

func uninstall() (string, error) {
	if hasSystemd() {
		_ = exec.Command("systemctl", "--user", "disable", "--now", unitName+".service").Run()
		_ = os.Remove(servicePath())
		_ = exec.Command("systemctl", "--user", "disable", "--now", unitName+".timer").Run()
		_ = os.Remove(timerPath())
		_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
		if hasCronEntry() {
			_ = removeCron()
		}
		return i18n.Tf("removed systemd user service %s.service", unitName), nil
	}
	if err := removeCron(); err != nil {
		return "", err
	}
	return i18n.T("removed crontab entry"), nil
}

func status() (string, error) {
	if hasSystemd() {
		out, err := exec.Command("systemctl", "--user", "is-active", unitName+".service").Output()
		state := strings.TrimSpace(string(out))
		if err != nil || state != "active" {
			if _, statErr := os.Stat(servicePath()); statErr == nil {
				return i18n.Tf("installed but not active\n  unit: %s", servicePath()), nil
			}
			return i18n.T("not installed"), nil
		}
		return i18n.Tf("installed and active\n  unit: %s", servicePath()), nil
	}
	if hasCronEntry() {
		return i18n.T("installed via crontab (fires at minute granularity)"), nil
	}
	return i18n.T("not installed"), nil
}

func installed() bool {
	if hasSystemd() {
		if _, err := os.Stat(servicePath()); err == nil {
			return true
		}
	}
	return hasCronEntry()
}

func restart() (string, error) {
	if hasSystemd() {
		if _, err := os.Stat(servicePath()); err != nil {
			return "", nil
		}
		if out, err := exec.Command("systemctl", "--user", "restart", unitName+".service").CombinedOutput(); err != nil {
			return "", fmt.Errorf("systemctl restart: %v (%s)", err, out)
		}
		return i18n.Tf("restarted systemd user service %s.service", unitName), nil
	}
	// A crontab entry spawns a fresh process each minute; nothing to restart.
	return "", nil
}

func installSystemd(exe string) (string, error) {
	if err := os.MkdirAll(unitDir(), 0o755); err != nil {
		return "", err
	}
	// Drop the legacy timer from older versions before enabling the service.
	_ = exec.Command("systemctl", "--user", "disable", "--now", unitName+".timer").Run()
	_ = os.Remove(timerPath())
	if err := os.WriteFile(servicePath(), []byte(serviceBody(exe)), 0o644); err != nil {
		return "", err
	}
	if out, err := exec.Command("systemctl", "--user", "daemon-reload").CombinedOutput(); err != nil {
		return "", fmt.Errorf("systemctl daemon-reload: %v (%s)", err, out)
	}
	if out, err := exec.Command("systemctl", "--user", "enable", "--now", unitName+".service").CombinedOutput(); err != nil {
		return "", fmt.Errorf("systemctl enable --now: %v (%s)", err, out)
	}
	return i18n.Tf("installed systemd user service %s.service\n  unit:   %s\n  action: %s remind daemon (resident scheduler)", unitName, servicePath(), exe), nil
}

// --- cron fallback --------------------------------------------------------

func readCrontab() string {
	out, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func hasCronEntry() bool { return strings.Contains(readCrontab(), cronMarker) }

func writeCrontab(content string) error {
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(content)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("crontab: %v (%s)", err, out)
	}
	return nil
}

func installCron(exe string) (string, error) {
	existing := readCrontab()
	line := fmt.Sprintf("* * * * * %s remind daemon --tick >/dev/null 2>&1 %s", exe, cronMarker)
	if strings.Contains(existing, cronMarker) {
		return i18n.T("crontab entry already present"), nil
	}
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	if err := writeCrontab(existing + line + "\n"); err != nil {
		return "", err
	}
	return i18n.Tf("installed crontab entry (every minute)\n  action: %s remind daemon --tick", exe), nil
}

func removeCron() error {
	lines := strings.Split(readCrontab(), "\n")
	var kept []string
	for _, l := range lines {
		if strings.Contains(l, cronMarker) {
			continue
		}
		kept = append(kept, l)
	}
	return writeCrontab(strings.Join(kept, "\n"))
}
