package fuzzy

// IsTypo reports whether candidate is a plausible typo of term — within a small, length-scaled
// Levenshtein distance. Terms shorter than 4 characters are never typo-matched: at that length too many unrelated words are a single edit apart (e.g. "car"/"cart"/"care") to match safely. Both inputs should already be normalized (e.g. lowercased) by the caller.
//
// Tolerance is 1 edit for 4–5 character terms and 2 for longer ones. The looser bound on longer terms also admits a single adjacent transposition — which Levenshtein scores as 2 edits — so common typos like "updaet" → "update" still match.
func IsTypo(term, candidate string) bool {
	if len(term) < 4 || len(candidate) < 4 {
		return false
	}
	maxDist := 1
	if len(term) >= 6 {
		maxDist = 2
	}
	// A length gap wider than the tolerance can't be closed within it — skip the distance computation.
	if abs(len(term)-len(candidate)) > maxDist {
		return false
	}
	return LevenshteinDistance(term, candidate) <= maxDist
}

// AnyTypo reports whether term is a typo of any of the candidates (see IsTypo).
func AnyTypo(term string, candidates []string) bool {
	for _, c := range candidates {
		if IsTypo(term, c) {
			return true
		}
	}
	return false
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
