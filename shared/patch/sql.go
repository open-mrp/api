package patch

import gosql "database/sql"

// StringToNullString maps a string Field to sql.NullString for repository updates.
func StringToNullString(f Field[string]) gosql.NullString {
	if f.IsClear() {
		return gosql.NullString{}
	}
	if f.IsSet() {
		val, _ := f.Value()
		return gosql.NullString{String: val, Valid: true}
	}
	return gosql.NullString{}
}

// OptionalStringToNullString maps a resolved string pointer (after service backfill) to sql.NullString.
func OptionalStringToNullString(s *string) gosql.NullString {
	if s == nil {
		return gosql.NullString{}
	}
	return gosql.NullString{String: *s, Valid: true}
}
