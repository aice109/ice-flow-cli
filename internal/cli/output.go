package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/aice109/ice-flow-cli/internal/store"
)

// colorEnabled is toggled once per invocation by setupColor.
var colorEnabled = true

func colorize(s, code string) string {
	if !colorEnabled {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func bold(s string) string   { return colorize(s, "1") }
func dim(s string) string    { return colorize(s, "2") }
func green(s string) string  { return colorize(s, "32") }
func yellow(s string) string { return colorize(s, "33") }
func red(s string) string    { return colorize(s, "31") }
func cyan(s string) string   { return colorize(s, "36") }

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// printTable renders headers and rows with aligned columns.
func printTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if len(headers) > 0 {
		fmt.Fprintln(w, strings.Join(headers, "\t"))
	}
	for _, r := range rows {
		fmt.Fprintln(w, strings.Join(r, "\t"))
	}
	w.Flush()
}

// printJSON marshals v as indented JSON.
func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// statusGlyph renders a task status marker.
func statusGlyph(status string) string {
	switch status {
	case store.StatusDoing:
		return yellow("◐")
	case store.StatusDone:
		return green("☑")
	default:
		return dim("☐")
	}
}

// humanDur renders a duration compactly ("5m", "3h", "2d").
func humanDur(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// relTime describes a time relative to now ("in 2h", "3d overdue").
func relTime(t, now time.Time) string {
	d := t.Sub(now)
	if d < 0 {
		return humanDur(d) + " overdue"
	}
	return "in " + humanDur(d)
}

// fmtLocal renders a time in the local timezone using layout.
func fmtLocal(t time.Time, layout string) string {
	return t.Local().Format(layout)
}

// fmtDue renders an optional due time as "2006-01-02 (in 3d)".
func fmtDue(due *time.Time, now time.Time) string {
	if due == nil {
		return dim("-")
	}
	base := fmtLocal(*due, "2006-01-02 15:04")
	rel := relTime(*due, now)
	if due.Before(now) {
		return red(base + " (" + rel + ")")
	}
	if due.Sub(now) < 24*time.Hour {
		return yellow(base + " (" + rel + ")")
	}
	return base + " (" + rel + ")"
}
