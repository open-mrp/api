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
