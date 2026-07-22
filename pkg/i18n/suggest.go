// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

// levenshtein returns the edit distance between a and b - the minimum
// number of single-character insertions, deletions, or substitutions
// needed to turn one into the other. Used only to suggest a likely
// intended key for a probable typo (see closestKey) - never on any render
// path, so a plain O(len(a)*len(b)) dynamic-programming table is more than
// fast enough for the translation-key-sized strings it's given.
func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 {
		return len(rb)
	}
	if len(rb) == 0 {
		return len(ra)
	}

	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			min := del
			if ins < min {
				min = ins
			}
			if sub < min {
				min = sub
			}
			curr[j] = min
		}
		prev, curr = curr, prev
	}
	return prev[len(rb)]
}

// closestKey returns the candidate closest to key by edit distance, and
// whether it's close enough to plausibly be what a typo of key was meant
// to be - "close enough" being at most a quarter of key's own length (a
// short key tolerates only a character or two of drift; a long one a bit
// more), so a key that's simply new/unrelated to anything in candidates
// isn't given a nonsensical suggestion. candidates is never mutated.
func closestKey(key string, candidates map[string]bool) (best string, ok bool) {
	threshold := len(key) / 4
	if threshold < 1 {
		threshold = 1
	}
	bestDist := threshold + 1
	for c := range candidates {
		d := levenshtein(key, c)
		if d < bestDist {
			bestDist = d
			best = c
		}
	}
	return best, bestDist <= threshold
}
