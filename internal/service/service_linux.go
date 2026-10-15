//go:build linux

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aice109/ice-flow-cli/internal/buildinfo"
)

const (
	unitName   = "iflow"
	cronMarker = "# ice-flow (" + buildinfo.BundleID + ")"
)

func unitDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user")
}

func servicePath() string { return filepath.Join(unitDir(), unitName+".service") }
func timerPath() string   { return filepath.Join(unitDir(), unitName+".timer") }

func serviceBody(exe string) string {
	return fmt.Sprintf(`[Unit]
Description=ice-flow reminder tick (%s)

[Service]
Type=oneshot
ExecStart=%s remind daemon --tick
`, Label(), exe)
}

func timerBody() string {
	return `[Unit]
Description=ice-flow reminder tick timer

[Timer]
OnBootSec=30s
OnUnitActiveSec=60s
Persistent=true

[Install]
WantedBy=timers.target
`
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
		_ = exec.Command("systemctl", "--user", "disable", "--now", unitName+".timer").Run()
		_ = os.Remove(timerPath())
		_ = os.Remove(servicePath())
		_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
		if hasCronEntry() {
			_ = removeCron()
		}
		return fmt.Sprintf("removed systemd user timer %s.timer", unitName), nil
	}
	if err := removeCron(); err != nil {
		return "", err
	}
	return "removed crontab entry", nil
}

func status() (string, error) {
	if hasSystemd() {
		out, err := exec.Command("systemctl", "--user", "is-active", unitName+".timer").Output()
		state := strings.TrimSpace(string(out))
		if err != nil || state != "active" {
			if _, statErr := os.Stat(timerPath()); statErr == nil {
				return fmt.Sprintf("installed but not active\n  unit: %s", timerPath()), nil
			}
			return "not installed", nil
		}
		return fmt.Sprintf("installed and active\n  unit: %s", timerPath()), nil
	}
	if hasCronEntry() {
		return "installed via crontab", nil
	}
	return "not installed", nil
}

func installSystemd(exe string) (string, error) {
	if err := os.MkdirAll(unitDir(), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(servicePath(), []byte(serviceBody(exe)), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(timerPath(), []byte(timerBody()), 0o644); err != nil {
		return "", err
	}
	if out, err := exec.Command("systemctl", "--user", "daemon-reload").CombinedOutput(); err != nil {
		return "", fmt.Errorf("systemctl daemon-reload: %v (%s)", err, out)
	}
	if out, err := exec.Command("systemctl", "--user", "enable", "--now", unitName+".timer").CombinedOutput(); err != nil {
		return "", fmt.Errorf("systemctl enable --now: %v (%s)", err, out)
	}
	return fmt.Sprintf("installed systemd user timer %s.timer\n  unit:   %s\n  action: %s remind daemon --tick (every 60s)", unitName, timerPath(), exe), nil
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
		return "crontab entry already present", nil
	}
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	if err := writeCrontab(existing + line + "\n"); err != nil {
		return "", err
	}
	return "installed crontab entry (every minute)\n  action: " + exe + " remind daemon --tick", nil
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
