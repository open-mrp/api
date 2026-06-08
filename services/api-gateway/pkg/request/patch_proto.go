package apirequest

import (
	"github.com/augno/api/shared/field"
	pb "github.com/augno/api/shared/proto/core"
)

// QuantityFieldToProto converts a quantity field to protobuf.
func QuantityFieldToProto(f field.Clearable[QuantityInput]) *pb.QuantityPatch {
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
