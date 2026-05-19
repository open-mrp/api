package patch

import (
	pb "github.com/augno/api/shared/proto/core"
)

// StringList is a type alias so slice type parameters work with patch fields.
type StringList []string

// StringFieldPtrToProto converts a string field pointer to protobuf. Nil means unset.
func StringFieldPtrToProto(f *Field[string]) *pb.StringPatch {
	return StringFieldToProto(Coalesce(f))
}

// StringFieldToProto converts a string field to protobuf. Returns nil when unset.
func StringFieldToProto(f Field[string]) *pb.StringPatch {
	if f.IsUnset() {
		return nil
	}
	if f.IsClear() {
		return &pb.StringPatch{Clear: true}
	}
	val, _ := f.Value()
	return &pb.StringPatch{Clear: false, Value: &val}
}

// StringFieldFromProto converts protobuf to a string field. Nil means unset.
func StringFieldFromProto(p *pb.StringPatch) Field[string] {
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

// StringListFieldToProto converts a string-list field to protobuf. Returns nil when unset.
func StringListFieldToProto(f Field[StringList]) *pb.StringListPatch {
	if f.IsUnset() {
		return nil
	}
	if f.IsClear() {
		return &pb.StringListPatch{Clear: true}
	}
	val, _ := f.Value()
	return &pb.StringListPatch{Clear: false, Value: []string(val)}
}

// StringListFieldFromProto converts protobuf to a string-list field.
func StringListFieldFromProto(p *pb.StringListPatch) Field[StringList] {
	if p == nil {
		return Unset[StringList]()
	}
	if p.Clear {
		return Clear[StringList]()
	}
	return Set(StringList(p.Value))
}

// SliceFieldToStringListField converts []string patch fields for proto list patches.
func SliceFieldToStringListField(f Field[[]string]) Field[StringList] {
	if f.IsUnset() {
		return Unset[StringList]()
	}
	if f.IsClear() {
		return Clear[StringList]()
	}
	v, _ := f.Value()
	return Set(StringList(v))
}

// StringListSliceFieldToProto converts a []string field to protobuf.
func StringListSliceFieldToProto(f Field[[]string]) *pb.StringListPatch {
	return StringListFieldToProto(SliceFieldToStringListField(f))
}

// StringListSliceFieldPtrToProto converts a []string field pointer to protobuf.
func StringListSliceFieldPtrToProto(f *Field[[]string]) *pb.StringListPatch {
	return StringListSliceFieldToProto(Coalesce(f))
}
