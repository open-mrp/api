package field

import (
	pb "github.com/augno/api/shared/proto/core"
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
