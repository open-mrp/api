package textutil

// Pluralize returns singular when count is 1 and plural otherwise.
func Pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}
