package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
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

// ansiRe matches SGR (color/style) escape sequences.
var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

// stripANSI removes color/style escape sequences from s.
func stripANSI(s string) string {
	if !strings.Contains(s, "\x1b[") {
		return s
	}
	return ansiRe.ReplaceAllString(s, "")
}

// displayWidth reports the number of terminal cells s occupies, ignoring ANSI
// escape sequences and counting East Asian wide runes as two cells.
func displayWidth(s string) int {
	w := 0
	for _, r := range stripANSI(s) {
		w += runeWidth(r)
	}
	return w
}

// runeWidth returns the cell width of a rune: 2 for East Asian wide and most
// emoji, 1 otherwise. Zero-width and control runes count as 1 to stay safe.
func runeWidth(r rune) int {
	switch {
	case r >= 0x1100 && r <= 0x115F, // Hangul Jamo
		r == 0x2329, r == 0x232A,
		r >= 0x2E80 && r <= 0x303E, // CJK radicals, Kangxi, CJK symbols
		r >= 0x3041 && r <= 0x33FF, // Hiragana .. CJK compatibility
		r >= 0x3400 && r <= 0x4DBF, // CJK extension A
		r >= 0x4E00 && r <= 0x9FFF, // CJK unified ideographs
		r >= 0xA000 && r <= 0xA4CF, // Yi
		r >= 0xAC00 && r <= 0xD7A3, // Hangul syllables
		r >= 0xF900 && r <= 0xFAFF, // CJK compatibility ideographs
		r >= 0xFE30 && r <= 0xFE6F, // CJK compatibility forms
		r >= 0xFF00 && r <= 0xFF60, // fullwidth forms
		r >= 0xFFE0 && r <= 0xFFE6,
		r >= 0x1F300 && r <= 0x1FAFF, // emoji
		r >= 0x20000 && r <= 0x3FFFD: // CJK extension B and beyond
		return 2
	}
	return 1
}

// writeTable renders headers and rows with aligned columns. Alignment is
// computed from visible width, so ANSI colors and wide (CJK) runes line up.
func writeTable(w io.Writer, headers []string, rows [][]string) {
	all := make([][]string, 0, len(rows)+1)
	if len(headers) > 0 {
		all = append(all, headers)
	}
	all = append(all, rows...)
	if len(all) == 0 {
		return
	}

	cols := 0
	for _, r := range all {
		if len(r) > cols {
			cols = len(r)
		}
	}
	widths := make([]int, cols)
	for _, r := range all {
		for i, c := range r {
			if cw := displayWidth(c); cw > widths[i] {
				widths[i] = cw
			}
		}
	}

	var b strings.Builder
	for _, r := range all {
		b.Reset()
		for i := 0; i < cols; i++ {
			cell := ""
			if i < len(r) {
				cell = r[i]
			}
			b.WriteString(cell)
			if i == cols-1 {
				break
			}
			pad := widths[i] - displayWidth(cell) + 2
			if pad < 0 {
				pad = 0
			}
			b.WriteString(strings.Repeat(" ", pad))
		}
		fmt.Fprintln(w, strings.TrimRight(b.String(), " "))
	}
}

// printTable renders a table to stdout.
func printTable(headers []string, rows [][]string) {
	writeTable(os.Stdout, headers, rows)
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

// humanDur renders a duration in unambiguous units ("12 min", "13 hr",
// "2 days"). "m" alone would be read as either minutes or months, so the unit
// is always spelled out.
func humanDur(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	switch {
	case d < time.Minute:
		return "<1 min"
	case d < time.Hour:
		return plural(int(d.Minutes()), "min")
	case d < 24*time.Hour:
		return plural(int(d.Hours()), "hr")
	default:
		return plural(int(d.Hours()/24), "day")
	}
}

func plural(n int, unit string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, unit)
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

// relTime describes a time relative to now, e.g. "in 13 hr" or "5 min overdue".
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

// fmtDue renders an optional due time as "<date> (<relative>)", colored by
// urgency: red when overdue, yellow when due within a day.
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
