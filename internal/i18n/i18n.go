// Package i18n provides a tiny message catalog for the CLI.
//
// English is the source language: user-facing messages are written in English
// in the code and looked up in a per-locale catalog. A missing entry falls back
// to the English text, so an incomplete translation degrades gracefully rather
// than printing an empty string or a key.
//
// The locale is resolved once, before the cobra command tree is built, so help
// text and flag descriptions can be translated too.
package i18n

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Supported locales.
const (
	English           = "en"
	ChineseSimplified = "zh-CN"
)

var (
	mu      sync.RWMutex
	current = English

	catalogs = map[string]map[string]string{
		ChineseSimplified: zhCN,
	}
)

// Locale returns the active locale.
func Locale() string {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// SetLocale switches the active locale. Unrecognized values select English.
func SetLocale(locale string) {
	if c := Canonical(locale); c != "" {
		locale = c
	} else {
		locale = English
	}
	mu.Lock()
	current = locale
	mu.Unlock()
}

// Supported lists the locales that have a translation catalog, English first.
func Supported() []string {
	out := make([]string, 0, len(catalogs)+1)
	out = append(out, English)
	for l := range catalogs {
		if l != English {
			out = append(out, l)
		}
	}
	sort.Strings(out[1:])
	return out
}

// Canonical normalizes a POSIX (zh_CN.UTF-8) or BCP-47 (zh-Hans) language tag to
// a supported locale. It returns "" for empty, "auto", "C"/"POSIX", or
// unsupported tags so callers can fall through to the next candidate.
func Canonical(tag string) string {
	s := strings.ToLower(strings.TrimSpace(tag))
	if s == "" || s == "auto" || s == "c" || s == "posix" {
		return ""
	}
	if i := strings.IndexAny(s, ".@"); i >= 0 { // strip .UTF-8 and @modifier
		s = s[:i]
	}
	s = strings.ReplaceAll(s, "_", "-")
	switch {
	case s == "zh" || strings.HasPrefix(s, "zh-"):
		return ChineseSimplified
	case s == "en" || strings.HasPrefix(s, "en-"):
		return English
	}
	return ""
}

// Detect picks a locale from the POSIX environment, defaulting to English.
func Detect(getenv func(string) string) string {
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if c := Canonical(getenv(k)); c != "" {
			return c
		}
	}
	return English
}

// T returns the translation of the English message s for the active locale.
func T(s string) string {
	if s == "" {
		return s
	}
	mu.RLock()
	cat := catalogs[current]
	mu.RUnlock()
	if cat == nil {
		return s
	}
	if v, ok := cat[s]; ok {
		return v
	}
	return s
}

// Tf translates the English format string s and applies args to the result.
func Tf(s string, args ...any) string {
	f := T(s)
	if len(args) == 0 {
		return f
	}
	return fmt.Sprintf(f, args...)
}
