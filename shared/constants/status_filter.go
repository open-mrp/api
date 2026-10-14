package constants

// IsActiveFilter is the `is_active` value a `statuses[]` filter selects, or nil when it selects every status
// (none given, or both). Repositories bind it as a nullable parameter:
//
//	AND (sqlc.narg(is_active) IS NULL OR t.is_active = sqlc.narg(is_active))
func IsActiveFilter[S ArchiveStatus | PauseStatus](statuses []S) *bool {
	var active, inactive bool
	for _, s := range statuses {
		if string(s) == string(ArchiveStatusActive) {
			active = true
		} else {
			inactive = true
		}
	}
	if active == inactive {
		return nil
	}
	return &active
}
