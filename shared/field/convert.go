package field

import (
	pb "github.com/augno/api/shared/proto/core"
)

// QuantityInput holds amount and unit for quantity patch fields.
type QuantityInput struct {
	Value  string
	UnitID string
}

// QuantityClearableFromProto converts protobuf to a quantity field.
func QuantityClearableFromProto(p *pb.QuantityPatch) Clearable[QuantityInput] {
	if p == nil {
		return Unset[QuantityInput]()
	}
	if p.Clear {
		return Clear[QuantityInput]()
	}
	if p.Value == nil || p.UnitId == nil {
		return Clear[QuantityInput]()
	}
	return Set(QuantityInput{Value: *p.Value, UnitID: *p.UnitId})
}
