package field

import gosql "database/sql"

// StringToNullString maps a string Clearable to sql.NullString for repository updates.
func StringToNullString(f Clearable[string]) gosql.NullString {
	if f.IsClear() {
		return gosql.NullString{}
	}
	if f.IsSet() {
		val, _ := f.Value()
		return gosql.NullString{String: val, Valid: true}
	}
	return gosql.NullString{}
}

// Int32ToNullInt32 maps an int32 Clearable to sql.NullInt32 for repository updates.
//
// An unset field maps to NULL like a cleared one, so callers that mean "leave it alone" must backfill from the existing row first, the same way StringToNullString's callers do.
func Int32ToNullInt32(f Clearable[int32]) gosql.NullInt32 {
	if f.IsClear() {
		return gosql.NullInt32{}
	}
	if f.IsSet() {
		val, _ := f.Value()
		return gosql.NullInt32{Int32: val, Valid: true}
	}
	return gosql.NullInt32{}
}
