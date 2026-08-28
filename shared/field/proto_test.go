package field

import (
	"reflect"
	"testing"
	"time"

	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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

func strPtr(s string) *string { return &s }

func TestStringClearableFromProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		in    *pb.StringPatch
		want  Clearable[string]
		value string
	}{
		{name: "nil is unset", in: nil, want: Unset[string]()},
		{name: "clear flag clears", in: &pb.StringPatch{Clear: true}, want: Clear[string]()},
		// A malformed patch (no clear flag, no value) is treated as a clear: the helper cannot
		// report an error, so a raw gRPC client that omits value NULLs the column.
		{name: "no clear flag and no value clears", in: &pb.StringPatch{Clear: false, Value: nil}, want: Clear[string]()},
		{name: "clear flag wins over a value", in: &pb.StringPatch{Clear: true, Value: strPtr("x")}, want: Clear[string]()},
		{name: "value sets", in: &pb.StringPatch{Clear: false, Value: strPtr("x")}, want: Set("x"), value: "x"},
		{name: "empty value sets blank, unlike the enum helper", in: &pb.StringPatch{Clear: false, Value: strPtr("")}, want: Set(""), value: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := StringClearableFromProto(tt.in)
			assertClearableState(t, got.IsUnset(), got.IsClear(), got.IsSet(), tt.want)
			if v, _ := got.Value(); v != tt.value {
				t.Fatalf("value: got %q, want %q", v, tt.value)
			}
		})
	}
}

func TestInt32ClearableFromProto(t *testing.T) {
	t.Parallel()

	val := int32(42)
	zero := int32(0)
	tests := []struct {
		name  string
		in    *pb.Int32Patch
		want  Clearable[int32]
		value int32
	}{
		{name: "nil is unset", in: nil, want: Unset[int32]()},
		{name: "clear flag clears", in: &pb.Int32Patch{Clear: true}, want: Clear[int32]()},
		// Same malformed-patch hazard as StringClearableFromProto: a missing value NULLs the
		// column (customer credit limits, lead times) instead of erroring.
		{name: "no clear flag and no value clears", in: &pb.Int32Patch{Clear: false, Value: nil}, want: Clear[int32]()},
		{name: "value sets", in: &pb.Int32Patch{Clear: false, Value: &val}, want: Set(val), value: val},
		{name: "zero value sets 0, not clear", in: &pb.Int32Patch{Clear: false, Value: &zero}, want: Set(zero), value: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Int32ClearableFromProto(tt.in)
			assertClearableState(t, got.IsUnset(), got.IsClear(), got.IsSet(), tt.want)
			if v, _ := got.Value(); v != tt.value {
				t.Fatalf("value: got %d, want %d", v, tt.value)
			}
		})
	}
}

func TestTimestampClearableFromProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   *pb.TimestampPatch
		want Clearable[time.Time]
	}{
		{name: "nil is unset", in: nil, want: Unset[time.Time]()},
		{name: "clear flag clears", in: &pb.TimestampPatch{Clear: true}, want: Clear[time.Time]()},
		{name: "no clear flag and no value clears", in: &pb.TimestampPatch{Clear: false, Value: nil}, want: Clear[time.Time]()},
		{name: "value sets", in: &pb.TimestampPatch{Clear: false, Value: timestamppb.New(refTime)}, want: Set(refTime)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := TimestampClearableFromProto(tt.in)
			assertClearableState(t, got.IsUnset(), got.IsClear(), got.IsSet(), tt.want)
			if v, ok := got.Value(); ok && !v.Equal(refTime) {
				t.Fatalf("value: got %v, want %v", v, refTime)
			}
		})
	}
}

// TestTimestampClearableFromProto_outOfRange pins that the helper does no range check: it has
// no error channel, so a timestamp a raw gRPC client cannot have meant passes straight through
// to a DATETIME column. Callers that accept untrusted patches must CheckValid first.
func TestTimestampClearableFromProto_outOfRange(t *testing.T) {
	t.Parallel()

	oob := &timestamppb.Timestamp{Seconds: 1 << 60}
	if err := oob.CheckValid(); err == nil {
		t.Fatal("fixture is no longer out of range")
	}

	got := TimestampClearableFromProto(&pb.TimestampPatch{Value: oob})
	v, ok := got.Value()
	if !ok {
		t.Fatalf("want a set field, got %+v", got)
	}
	if v.Year() <= 9999 {
		t.Fatalf("want the unchecked year through, got %v", v)
	}
}

func TestStringListClearableFromProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		in    *pb.StringListPatch
		want  Clearable[StringList]
		value StringList
	}{
		{name: "nil is unset", in: nil, want: Unset[StringList]()},
		{name: "clear flag clears", in: &pb.StringListPatch{Clear: true}, want: Clear[StringList]()},
		// The list helper diverges from the scalar ones: a nil value is a SET empty list, not a
		// clear, so a malformed patch empties the list rather than NULLing the column.
		{name: "no clear flag and no value sets empty", in: &pb.StringListPatch{Clear: false, Value: nil}, want: Set(StringList(nil)), value: nil},
		{name: "value sets", in: &pb.StringListPatch{Clear: false, Value: []string{"a", "b"}}, want: Set(StringList{"a", "b"}), value: StringList{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := StringListClearableFromProto(tt.in)
			assertClearableState(t, got.IsUnset(), got.IsClear(), got.IsSet(), tt.want)
			if v, _ := got.Value(); !reflect.DeepEqual(v, tt.value) {
				t.Fatalf("value: got %#v, want %#v", v, tt.value)
			}
		})
	}
}

func TestQuantityClearableFromProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		in    *pb.QuantityPatch
		want  Clearable[QuantityInput]
		value QuantityInput
	}{
		{name: "nil is unset", in: nil, want: Unset[QuantityInput]()},
		{name: "clear flag clears", in: &pb.QuantityPatch{Clear: true}, want: Clear[QuantityInput]()},
		// A half-filled patch clears rather than erroring, so a client that sends an amount
		// without a unit wipes the column it meant to write.
		{name: "value without unit clears", in: &pb.QuantityPatch{Value: strPtr("10")}, want: Clear[QuantityInput]()},
		{name: "unit without value clears", in: &pb.QuantityPatch{UnitId: strPtr("unit_1")}, want: Clear[QuantityInput]()},
		{name: "both set", in: &pb.QuantityPatch{Value: strPtr("10"), UnitId: strPtr("unit_1")}, want: Set(QuantityInput{Value: "10", UnitID: "unit_1"}), value: QuantityInput{Value: "10", UnitID: "unit_1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := QuantityClearableFromProto(tt.in)
			assertClearableState(t, got.IsUnset(), got.IsClear(), got.IsSet(), tt.want)
			if v, _ := got.Value(); v != tt.value {
				t.Fatalf("value: got %+v, want %+v", v, tt.value)
			}
		})
	}
}

func TestEnumClearableFromProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		in    *pb.StringPatch
		want  Clearable[testEnum]
		value testEnum
	}{
		{name: "nil is unset", in: nil, want: Unset[testEnum]()},
		{name: "clear flag clears", in: &pb.StringPatch{Clear: true}, want: Clear[testEnum]()},
		{name: "no value clears", in: &pb.StringPatch{Clear: false, Value: nil}, want: Clear[testEnum]()},
		{name: "empty value clears", in: &pb.StringPatch{Clear: false, Value: strPtr("")}, want: Clear[testEnum]()},
		{name: "value sets", in: &pb.StringPatch{Clear: false, Value: strPtr("tag")}, want: Set(testEnum("tag")), value: "tag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := EnumClearableFromProto[testEnum](tt.in)
			assertClearableState(t, got.IsUnset(), got.IsClear(), got.IsSet(), tt.want)
			if v, _ := got.Value(); v != tt.value {
				t.Fatalf("value: got %q, want %q", v, tt.value)
			}
		})
	}
}

// TestClearable_protoRoundTrip covers the shape the gateway actually produces: to-proto on the
// way in, from-proto on the service side. Unset must survive as unset, or an omitted PATCH field
// turns into a clear one hop later.
func TestClearable_protoRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("string", func(t *testing.T) {
		t.Parallel()
		for name, want := range map[string]Clearable[string]{
			"unset": Unset[string](),
			"clear": Clear[string](),
			"set":   Set("value"),
			"blank": Set(""),
		} {
			got := StringClearableFromProto(StringClearableToProto(want))
			assertClearableState(t, got.IsUnset(), got.IsClear(), got.IsSet(), want)
			gv, _ := got.Value()
			wv, _ := want.Value()
			if gv != wv {
				t.Fatalf("%s: value got %q, want %q", name, gv, wv)
			}
		}
	})

	t.Run("int32", func(t *testing.T) {
		t.Parallel()
		for name, want := range map[string]Clearable[int32]{
			"unset": Unset[int32](),
			"clear": Clear[int32](),
			"set":   Set(int32(9)),
			"zero":  Set(int32(0)),
		} {
			got := Int32ClearableFromProto(Int32ClearableToProto(want))
			assertClearableState(t, got.IsUnset(), got.IsClear(), got.IsSet(), want)
			gv, _ := got.Value()
			wv, _ := want.Value()
			if gv != wv {
				t.Fatalf("%s: value got %d, want %d", name, gv, wv)
			}
		}
	})

	t.Run("timestamp", func(t *testing.T) {
		t.Parallel()
		for name, want := range map[string]Clearable[time.Time]{
			"unset": Unset[time.Time](),
			"clear": Clear[time.Time](),
			"set":   Set(refTime),
		} {
			got := TimestampClearableFromProto(TimestampClearableToProto(want))
			assertClearableState(t, got.IsUnset(), got.IsClear(), got.IsSet(), want)
			gv, ok := got.Value()
			if wv, wok := want.Value(); wok != ok || (ok && !gv.Equal(wv)) {
				t.Fatalf("%s: value got %v, want %v", name, gv, wv)
			}
		}
	})

	t.Run("string list", func(t *testing.T) {
		t.Parallel()
		for name, want := range map[string]Clearable[StringList]{
			"unset": Unset[StringList](),
			"clear": Clear[StringList](),
			"set":   Set(StringList{"a", "b"}),
		} {
			got := StringListClearableFromProto(StringListClearableToProto(want))
			assertClearableState(t, got.IsUnset(), got.IsClear(), got.IsSet(), want)
			gv, _ := got.Value()
			wv, _ := want.Value()
			if !reflect.DeepEqual(gv, wv) {
				t.Fatalf("%s: value got %#v, want %#v", name, gv, wv)
			}
		}
	})

	t.Run("enum", func(t *testing.T) {
		t.Parallel()
		for name, want := range map[string]Clearable[testEnum]{
			"unset": Unset[testEnum](),
			"clear": Clear[testEnum](),
			"set":   Set(testEnum("tag")),
		} {
			got := EnumClearableFromProto[testEnum](EnumClearableToProto(want))
			assertClearableState(t, got.IsUnset(), got.IsClear(), got.IsSet(), want)
			gv, _ := got.Value()
			wv, _ := want.Value()
			if gv != wv {
				t.Fatalf("%s: value got %q, want %q", name, gv, wv)
			}
		}
	})
}

// TestSliceClearableToStringListClearable_preservesState guards the adapter used by the
// []string patch path: only the wrapper changes, never the state.
func TestSliceClearableToStringListClearable_preservesState(t *testing.T) {
	t.Parallel()

	if got := SliceClearableToStringListClearable(Unset[[]string]()); !got.IsUnset() {
		t.Errorf("unset: got %+v", got)
	}
	if got := SliceClearableToStringListClearable(Clear[[]string]()); !got.IsClear() {
		t.Errorf("clear: got %+v", got)
	}
	got := SliceClearableToStringListClearable(Set([]string{"a"}))
	v, ok := got.Value()
	if !ok || !reflect.DeepEqual(v, StringList{"a"}) {
		t.Errorf("set: got %#v ok=%v", v, ok)
	}
}

func assertClearableState[T any](t *testing.T, unset, clear, set bool, want Clearable[T]) {
	t.Helper()
	if unset != want.IsUnset() || clear != want.IsClear() || set != want.IsSet() {
		t.Fatalf("state: got unset=%v clear=%v set=%v, want unset=%v clear=%v set=%v",
			unset, clear, set, want.IsUnset(), want.IsClear(), want.IsSet())
	}
}
