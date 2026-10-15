// Package fuzzy implements a scored subsequence matcher inspired by fzf.
//
// A pattern matches a target when the pattern's characters appear in order
// anywhere in the target. Matches earn bonuses for landing on word boundaries
// (start of string, after a separator, or a camelCase hump) and for being
// consecutive, and pay a penalty for the gaps they skip. The returned
// positions let callers highlight the matched characters.
package fuzzy

import (
	"strings"
	"unicode"
)

// Result is the result of a successful pattern/target comparison.
type Result struct {
	// Score ranks matches; higher is better.
	Score int
	// Positions are the target rune indices that matched, in order.
	Positions []int
}

const (
	scoreMatch       = 16
	bonusBoundary    = 10
	bonusCamel       = 8
	bonusConsecutive = 10
	penaltyGap       = 2
	penaltyLeading   = 1
	neg              = -1 << 30
)

// Match scores pattern against target. The second return value reports
// whether the pattern is a subsequence of the target at all.
func Match(pattern, target string) (Result, bool) {
	p := []rune(strings.ToLower(strings.TrimSpace(pattern)))
	if len(p) == 0 {
		return Result{Score: 0}, true
	}
	orig := []rune(target)
	low := []rune(strings.ToLower(target))
	n, m := len(low), len(p)
	if m > n {
		return Result{}, false
	}

	charBonus := func(i int) int {
		b := 0
		if i == 0 {
			return bonusBoundary
		}
		prev := orig[i-1]
		if isSeparator(prev) {
			b += bonusBoundary
		}
		if unicode.IsLower(prev) && unicode.IsUpper(orig[i]) {
			b += bonusCamel
		}
		return b
	}

	dp := make([][]int, m)
	from := make([][]int, m)
	for j := range dp {
		dp[j] = make([]int, n)
		from[j] = make([]int, n)
		for i := range dp[j] {
			dp[j][i] = neg
			from[j][i] = -1
		}
	}

	for i := 0; i < n; i++ {
		if low[i] == p[0] {
			dp[0][i] = scoreMatch + charBonus(i) - i*penaltyLeading
		}
	}
	for j := 1; j < m; j++ {
		for i := j; i < n; i++ {
			if low[i] != p[j] {
				continue
			}
			best, bestFrom := neg, -1
			for k := j - 1; k < i; k++ {
				if dp[j-1][k] == neg {
					continue
				}
				sc := dp[j-1][k]
				if k == i-1 {
					sc += bonusConsecutive
				} else {
					sc -= (i - k - 1) * penaltyGap
				}
				if sc > best {
					best, bestFrom = sc, k
				}
			}
			if best == neg {
				continue
			}
			dp[j][i] = best + scoreMatch + charBonus(i)
			from[j][i] = bestFrom
		}
	}

	bestEnd, bestScore := -1, neg
	for i := 0; i < n; i++ {
		if dp[m-1][i] > bestScore {
			bestScore, bestEnd = dp[m-1][i], i
		}
	}
	if bestEnd == -1 {
		return Result{}, false
	}
	pos := make([]int, m)
	for j, i := m-1, bestEnd; j >= 0; j-- {
		pos[j] = i
		i = from[j][i]
	}
	return Result{Score: bestScore, Positions: pos}, true
}

// Score reports only the score of the best match, or (0,false).
func Score(pattern, target string) (int, bool) {
	m, ok := Match(pattern, target)
	return m.Score, ok
}

// Highlight wraps the matched runes of target in a pair of ANSI markers.
func Highlight(target string, positions []int, prefix, suffix string) string {
	if len(positions) == 0 {
		return target
	}
	set := make(map[int]bool, len(positions))
	for _, p := range positions {
		set[p] = true
	}
	var b strings.Builder
	for i, r := range []rune(target) {
		if set[i] {
			b.WriteString(prefix)
			b.WriteRune(r)
			b.WriteString(suffix)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isSeparator(r rune) bool {
	switch r {
	case ' ', '\t', '-', '_', '.', '/', '\\', ':', ',', '@', '+', '(', ')', '[', ']', '{', '}', '|', '>', '<', '=':
		return true
	}
	return false
}
