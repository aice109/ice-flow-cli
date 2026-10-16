package update

import "testing"

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"v1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.2.0", "1.10.0", -1},
		{"2.0.0", "1.9.9", 1},
		{"1.0.0", "1.0.0-rc1", -1},
	}
	for _, c := range cases {
		if got := CompareSemver(c.a, c.b); got != c.want {
			t.Errorf("CompareSemver(%q,%q)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestAssetName(t *testing.T) {
	got := AssetName("v1.2.3", "darwin", "arm64")
	want := "iceflow_1.2.3_darwin_arm64.tar.gz"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestChecksumFor(t *testing.T) {
	sums := "abc123  iceflow_1.2.3_darwin_arm64.tar.gz\ndef456  iceflow_1.2.3_linux_amd64.tar.gz\n"
	got, err := checksumFor(sums, "iceflow_1.2.3_linux_amd64.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	if got != "def456" {
		t.Errorf("got %q", got)
	}
	if _, err := checksumFor(sums, "missing"); err == nil {
		t.Error("expected error for missing asset")
	}
}
