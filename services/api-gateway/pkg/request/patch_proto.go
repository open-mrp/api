package apirequest

import (
	"github.com/augno/api/shared/patch"
	pb "github.com/augno/api/shared/proto/core"
)

// QuantityFieldPtrToProto converts a quantity field pointer to protobuf.
func QuantityFieldPtrToProto(f *patch.Field[QuantityInput]) *pb.QuantityPatch {
	return QuantityFieldToProto(patch.Coalesce(f))
}

// QuantityFieldToProto converts a quantity field to protobuf.
func QuantityFieldToProto(f patch.Field[QuantityInput]) *pb.QuantityPatch {
	if f.IsUnset() {
		return nil
	}
	if f.IsClear() {
		return &pb.QuantityPatch{Clear: true}
	}
	v, _ := f.Value()
	return &pb.QuantityPatch{
		Clear:  false,
		Value:  &v.Value,
		UnitId: &v.UnitID,
	}
}
