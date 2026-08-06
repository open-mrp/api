package field

import "testing"

type testEnum string

func TestEnumClearableToProto(t *testing.T) {
	t.Parallel()

	if got := EnumClearableToProto(Unset[testEnum]()); got != nil {
		t.Errorf("unset should convert to nil, got %+v", got)
	}

	if got := EnumClearableToProto(Clear[testEnum]()); got == nil || !got.Clear {
		t.Errorf("clear should convert to a clearing patch, got %+v", got)
	}

	// An explicitly set empty value is never a valid enum value and is treated
	// as a clear (spreadsheet-driven clients send "" for a blank cell).
	if got := EnumClearableToProto(Set(testEnum(""))); got == nil || !got.Clear {
		t.Errorf("set empty value should convert to a clearing patch, got %+v", got)
	}

	got := EnumClearableToProto(Set(testEnum("tag")))
	if got == nil || got.Clear || got.Value == nil || *got.Value != "tag" {
		t.Errorf("set value should convert to a value patch, got %+v", got)
	}
}
