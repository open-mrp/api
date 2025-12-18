package fuzzy

// LevenshteinDistance computes the Levenshtein edit distance between two strings.
// The implementation is iterative and uses O(min(m,n)) additional memory.
func LevenshteinDistance(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Ensure that a is the shorter string to reduce memory usage
	if len(a) > len(b) {
		a, b = b, a
	}

	prev := make([]int, len(a)+1)
	curr := make([]int, len(a)+1)

	for i := 0; i <= len(a); i++ {
		prev[i] = i
	}

	for j := 1; j <= len(b); j++ {
		curr[0] = j
		bj := b[j-1]
		for i := 1; i <= len(a); i++ {
			cost := 0
			if a[i-1] != bj {
				cost = 1
			}
			deletion := prev[i] + 1
			insertion := curr[i-1] + 1
			substitution := prev[i-1] + cost

			// Take min of deletion, insertion, substitution
			if deletion < insertion {
				if deletion < substitution {
					curr[i] = deletion
				} else {
					curr[i] = substitution
				}
			} else {
				if insertion < substitution {
					curr[i] = insertion
				} else {
					curr[i] = substitution
				}
			}
		}
		prev, curr = curr, prev
	}

	return prev[len(a)]
}

// FindClosestByLevenshtein returns the closest match to target from candidates and its distance.
// If candidates is empty, it returns ("", 0).
func FindClosestByLevenshtein(target string, candidates []string) (string, int) {
	best := ""
	bestDist := 0
	first := true
	for _, c := range candidates {
		d := LevenshteinDistance(target, c)
		if first || d < bestDist {
			best = c
			bestDist = d
			first = false
		}
	}
	return best, bestDist
}
