package constants

import "testing"

// A stub's type must name a real resource, so clients can fetch the full record it points at.
func TestRecordType_EveryValueIsARegisteredObjectType(t *testing.T) {
	t.Parallel()
	for _, v := range RecordType("").EnumValues() {
		if !RecordType(v).ObjectType().IsValid() {
			t.Errorf("record type %q is not a registered object type", v)
		}
	}
}
