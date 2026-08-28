package db

import "testing"

func TestNullInt64Ptr(t *testing.T) {
	t.Parallel()
	if got := NullInt64Ptr(nil); got.Valid {
		t.Fatalf("nil should bind NULL, got %+v", got)
	}
	zero := int64(0)
	if got := NullInt64Ptr(&zero); !got.Valid || got.Int64 != 0 {
		t.Fatalf("zero must bind as a value, not NULL, got %+v", got)
	}
	n := int64(-42)
	if got := NullInt64Ptr(&n); !got.Valid || got.Int64 != -42 {
		t.Fatalf("NullInt64Ptr(-42) = %+v, want {-42 true}", got)
	}
}

func TestInt64FromInterface(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		value  any
		want   int64
		wantOk bool
	}{
		{name: "nil", value: nil, want: 0, wantOk: false},
		{name: "int64", value: int64(7), want: 7, wantOk: true},
		{name: "negative int64", value: int64(-7), want: -7, wantOk: true},
		{name: "int", value: 7, want: 7, wantOk: true},
		{name: "int32", value: int32(7), want: 7, wantOk: true},
		{name: "int64 max", value: int64(1<<63 - 1), want: 1<<63 - 1, wantOk: true},
		{name: "digit bytes", value: []byte("1234"), want: 1234, wantOk: true},
		{name: "zero bytes", value: []byte("0"), want: 0, wantOk: true},
		{name: "leading zero bytes", value: []byte("007"), want: 7, wantOk: true},
		{name: "empty bytes", value: []byte{}, want: 0, wantOk: false},
		// A wrong number is worse than no number: anything that is not a plain digit run is
		// refused rather than half-parsed.
		{name: "decimal text bytes", value: []byte("12.5"), want: 0, wantOk: false},
		{name: "signed text bytes", value: []byte("+12"), want: 0, wantOk: false},
		{name: "negative text bytes", value: []byte("-12"), want: 0, wantOk: false},
		{name: "uint64", value: uint64(12), want: 0, wantOk: false},
		{name: "space padded bytes", value: []byte(" 12"), want: 0, wantOk: false},
		{name: "non-numeric bytes", value: []byte("abc"), want: 0, wantOk: false},
		{name: "string", value: "12", want: 0, wantOk: false},
		{name: "float64", value: 12.0, want: 0, wantOk: false},
		{name: "bool", value: true, want: 0, wantOk: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := Int64FromInterface(tt.value)
			if got != tt.want || ok != tt.wantOk {
				t.Fatalf("Int64FromInterface(%#v) = (%d, %v), want (%d, %v)", tt.value, got, ok, tt.want, tt.wantOk)
			}
		})
	}
}
