package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTarRoundTrip(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	want := "lang = \"en\"\n"
	if err := os.WriteFile(filepath.Join(src, backupConfigName), []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, backupDBName), []byte("SQLite format 3\x00"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, backupManifestName), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	archive := filepath.Join(dir, "backup.tar.gz")
	if err := writeTarGz(archive, src, []string{backupManifestName, backupConfigName, backupDBName}); err != nil {
		t.Fatalf("writeTarGz: %v", err)
	}

	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := extractTarGz(archive, out); err != nil {
		t.Fatalf("extractTarGz: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(out, backupConfigName))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("config.toml = %q, want %q", got, want)
	}
}

func TestExtractAcceptsLegacyDBName(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, backupDBLegacyName), []byte("legacy"), 0o644); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(dir, "legacy.tar.gz")
	if err := writeTarGz(archive, src, []string{backupDBLegacyName}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := extractTarGz(archive, out); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(out, backupDBName))
	if err != nil {
		t.Fatalf("legacy db was not renamed: %v", err)
	}
	if string(got) != "legacy" {
		t.Errorf("db = %q, want legacy", got)
	}
}
