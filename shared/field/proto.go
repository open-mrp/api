package field

import (
	"time"

	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// StringList is a type alias so slice type parameters work with patch fields.
type StringList []string

// StringClearableToProto converts a string field to protobuf. Returns nil when unset.
func StringClearableToProto(f Clearable[string]) *pb.StringPatch {
	if f.IsUnset() {
		return nil
	}
	if f.IsClear() {
		return &pb.StringPatch{Clear: true}
	}
	val, _ := f.Value()
	return &pb.StringPatch{Clear: false, Value: &val}
}

// EnumClearableToProto converts a string-enum field to protobuf. Returns nil when
// unset. An explicitly set empty value is treated as a clear — the empty string is
// never a valid enum value, and spreadsheet-driven clients send "" for a blank cell.
func EnumClearableToProto[T ~string](f Clearable[T]) *pb.StringPatch {
	if f.IsUnset() {
		return nil
	}
	val, _ := f.Value()
	if f.IsClear() || val == "" {
		return &pb.StringPatch{Clear: true}
	}
	s := string(val)
	return &pb.StringPatch{Clear: false, Value: &s}
}

// StringClearableFromProto converts protobuf to a string field. Nil means unset.
func StringClearableFromProto(p *pb.StringPatch) Clearable[string] {
	if p == nil {
		return Unset[string]()
	}
	if p.Clear {
		return Clear[string]()
	}
	if p.Value == nil {
		return Clear[string]()
	}
	return Set(*p.Value)
}

// StringListClearableToProto converts a string-list field to protobuf. Returns nil when unset.
func StringListClearableToProto(f Clearable[StringList]) *pb.StringListPatch {
	if f.IsUnset() {
		return nil
	}
	if f.IsClear() {
		return &pb.StringListPatch{Clear: true}
	}
	val, _ := f.Value()
	return &pb.StringListPatch{Clear: false, Value: []string(val)}
}

// StringListClearableFromProto converts protobuf to a string-list field.
func StringListClearableFromProto(p *pb.StringListPatch) Clearable[StringList] {
	if p == nil {
		return Unset[StringList]()
	}
	if p.Clear {
		return Clear[StringList]()
	}
	return Set(StringList(p.Value))
}

// SliceClearableToStringListClearable converts []string patch fields for proto list patches.
func SliceClearableToStringListClearable(f Clearable[[]string]) Clearable[StringList] {
	if f.IsUnset() {
		return Unset[StringList]()
	}
	if f.IsClear() {
		return Clear[StringList]()
	}
	v, _ := f.Value()
	return Set(StringList(v))
}

// StringListSliceClearableToProto converts a []string field to protobuf.
func StringListSliceClearableToProto(f Clearable[[]string]) *pb.StringListPatch {
	return StringListClearableToProto(SliceClearableToStringListClearable(f))
}

// TimestampClearableToProto converts a time.Time field to protobuf. Returns nil when unset.
func TimestampClearableToProto(f Clearable[time.Time]) *pb.TimestampPatch {
	if f.IsUnset() {
		return nil
	}
	if f.IsClear() {
		return &pb.TimestampPatch{Clear: true}
	}
	val, _ := f.Value()
	return &pb.TimestampPatch{Clear: false, Value: timestamppb.New(val)}
}

// TimestampClearableFromProto converts protobuf to a time.Time field. Nil means unset.
func TimestampClearableFromProto(p *pb.TimestampPatch) Clearable[time.Time] {
	if p == nil {
		return Unset[time.Time]()
	}
	if p.Clear || p.Value == nil {
		return Clear[time.Time]()
	}
	return Set(p.Value.AsTime())
}

// Int32ClearableToProto converts an int32 clearable field to protobuf. Returns nil when unset.
func Int32ClearableToProto(f Clearable[int32]) *pb.Int32Patch {
	if f.IsUnset() {
		return nil
	}
	if f.IsClear() {
		return &pb.Int32Patch{Clear: true}
	}
	val, _ := f.Value()
	return &pb.Int32Patch{Clear: false, Value: &val}
}

// Int32ClearableFromProto converts an int32 patch back to a clearable field.
func Int32ClearableFromProto(p *pb.Int32Patch) Clearable[int32] {
	if p == nil {
		return Unset[int32]()
	}
	if p.Clear || p.Value == nil {
		return Clear[int32]()
	}
	return Set(*p.Value)
}
