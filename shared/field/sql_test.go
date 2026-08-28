package field

import (
	gosql "database/sql"
	"testing"
)

// TestStringToNullString pins the contract callers most often get wrong: an UNSET field maps to
// NULL exactly like a cleared one, so a caller that skips the backfill NULLs a column the client
// never mentioned.
func TestStringToNullString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   Clearable[string]
		want gosql.NullString
	}{
		{"unset writes NULL", Unset[string](), gosql.NullString{}},
		{"clear writes NULL", Clear[string](), gosql.NullString{}},
		{"set writes value", Set("abc"), gosql.NullString{String: "abc", Valid: true}},
		{"set blank writes empty string, not NULL", Set(""), gosql.NullString{String: "", Valid: true}},
		{"backfilled unset writes the existing value", Unset[string]().BackfillUnset("existing"), gosql.NullString{String: "existing", Valid: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := StringToNullString(tt.in); got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestInt32ToNullInt32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   Clearable[int32]
		want gosql.NullInt32
	}{
		{"unset writes NULL", Unset[int32](), gosql.NullInt32{}},
		{"clear writes NULL", Clear[int32](), gosql.NullInt32{}},
		{"set writes value", Set(int32(7)), gosql.NullInt32{Int32: 7, Valid: true}},
		{"set zero writes 0, not NULL", Set(int32(0)), gosql.NullInt32{Int32: 0, Valid: true}},
		{"backfilled unset writes the existing value", Unset[int32]().BackfillUnset(5), gosql.NullInt32{Int32: 5, Valid: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Int32ToNullInt32(tt.in); got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestToNull_unsetIsIndistinguishableFromClear states the shared property directly: neither
// helper can tell "leave it alone" from "set it to NULL", which is why every call site must
// backfill from the existing row first.
func TestToNull_unsetIsIndistinguishableFromClear(t *testing.T) {
	t.Parallel()

	if StringToNullString(Unset[string]()) != StringToNullString(Clear[string]()) {
		t.Fatal("unset and clear must map to the same NullString")
	}
	if Int32ToNullInt32(Unset[int32]()) != Int32ToNullInt32(Clear[int32]()) {
		t.Fatal("unset and clear must map to the same NullInt32")
	}
}
