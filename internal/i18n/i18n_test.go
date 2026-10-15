package i18n

import "testing"

func TestCanonical(t *testing.T) {
	cases := map[string]string{
		"zh_CN.UTF-8": ChineseSimplified,
		"zh-cn":       ChineseSimplified,
		"zh":          ChineseSimplified,
		"zh-Hans":     ChineseSimplified,
		"zh@latin":    ChineseSimplified,
		"en_US":       English,
		"en":          English,
		"  EN-gb ":    English,
		"auto":        "",
		"":            "",
		"C":           "",
		"POSIX":       "",
		"de_DE":       "",
	}
	for in, want := range cases {
		if got := Canonical(in); got != want {
			t.Errorf("Canonical(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDetectPrefersLCAll(t *testing.T) {
	env := map[string]string{
		"LC_ALL":      "en_US.UTF-8",
		"LC_MESSAGES": "zh_CN.UTF-8",
		"LANG":        "de_DE.UTF-8",
	}
	got := Detect(func(k string) string { return env[k] })
	if got != English {
		t.Errorf("Detect = %q, want %q", got, English)
	}

	// An unsupported LC_ALL falls through to the next variable.
	env["LC_ALL"] = "de_DE.UTF-8"
	env["LC_MESSAGES"] = "zh_CN.UTF-8"
	if got := Detect(func(k string) string { return env[k] }); got != ChineseSimplified {
		t.Errorf("Detect = %q, want %q", got, ChineseSimplified)
	}

	if got := Detect(func(string) string { return "" }); got != English {
		t.Errorf("Detect(empty) = %q, want %q", got, English)
	}
}

func TestTranslateAndFallback(t *testing.T) {
	orig := Locale()
	t.Cleanup(func() { SetLocale(orig) })

	SetLocale(English)
	if got := T("Flags:"); got != "Flags:" {
		t.Errorf("English T = %q, want the source string", got)
	}

	SetLocale(ChineseSimplified)
	if got := T("Flags:"); got != "标志：" {
		t.Errorf("zh T = %q", got)
	}
	if got := Tf("added task %s", "#7"); got != "已添加任务 #7" {
		t.Errorf("zh Tf = %q", got)
	}
	// Unknown messages fall back to the English source.
	if got := T("some string with no translation"); got != "some string with no translation" {
		t.Errorf("fallback T = %q", got)
	}

	// Unrecognized locales select English rather than breaking output.
	SetLocale("de-DE")
	if got := Locale(); got != English {
		t.Errorf("Locale after unsupported = %q, want %q", got, English)
	}
}

func TestCatalogCoversEverySupportedLocale(t *testing.T) {
	for _, l := range Supported() {
		if l != English && len(catalogs[l]) == 0 {
			t.Errorf("locale %q has no catalog", l)
		}
	}
}
