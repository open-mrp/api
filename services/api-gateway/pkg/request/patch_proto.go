package apirequest

import (
	"github.com/augno/api/shared/field"
	pb "github.com/augno/api/shared/proto/core"
)

// QuantityFieldToProto converts a clearable quantity request field into its patch representation: an omitted field yields nil so the stored value is left alone, an explicit null yields a clear instruction, and a supplied object yields the new value and unit.
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
