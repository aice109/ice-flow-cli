package fuzzy

import "testing"

func TestMatchSubsequence(t *testing.T) {
	if _, ok := Match("fltdr", "flutter doctor"); !ok {
		t.Fatal("expected subsequence match for fltdr -> flutter doctor")
	}
	if _, ok := Match("xyz", "flutter"); ok {
		t.Fatal("did not expect a match")
	}
	if _, ok := Match("", "anything"); !ok {
		t.Fatal("empty pattern should match")
	}
	if _, ok := Match("toolongpattern", "ab"); ok {
		t.Fatal("pattern longer than target should not match")
	}
}

func TestMatchPositions(t *testing.T) {
	m, ok := Match("doctor", "flutter doctor")
	if !ok {
		t.Fatal("expected match")
	}
	if len(m.Positions) != 6 {
		t.Fatalf("expected 6 positions, got %d", len(m.Positions))
	}
	for i := 1; i < len(m.Positions); i++ {
		if m.Positions[i] <= m.Positions[i-1] {
			t.Fatalf("positions not increasing: %v", m.Positions)
		}
	}
}

func TestExactOutranksSubstring(t *testing.T) {
	exact, _ := Match("doctor", "doctor")
	suffix, _ := Match("doctor", "flutter doctor")
	if exact.Score <= suffix.Score {
		t.Errorf("exact match score %d should beat suffix match score %d", exact.Score, suffix.Score)
	}
}

func TestBoundaryBonusOutranksMidWord(t *testing.T) {
	boundary, OK := Match("get", "pub get")
	mid, _ := Match("get", "forget")
	if !OK {
		t.Fatal("expected match")
	}
	if boundary.Score <= mid.Score {
		t.Errorf("boundary match %d should beat mid-word match %d", boundary.Score, mid.Score)
	}
}

func TestHighlight(t *testing.T) {
	got := Highlight("flutter doctor", []int{0}, "[", "]")
	want := "[f]lutter doctor"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}
