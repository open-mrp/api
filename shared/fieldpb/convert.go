package fieldpb

import (
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/apikit/field"
)

// QuantityInput holds amount and unit for quantity patch fields.
type QuantityInput struct {
	Value  string
	UnitID string
}

// QuantityClearableFromProto converts protobuf to a quantity field.
func QuantityClearableFromProto(p *pb.QuantityPatch) field.Clearable[QuantityInput] {
	if p == nil {
		return field.Unset[QuantityInput]()
	}
	if p.Clear {
		return field.Clear[QuantityInput]()
	}
	if p.Value == nil || p.UnitId == nil {
		return field.Clear[QuantityInput]()
	}
	return field.Set(QuantityInput{Value: *p.Value, UnitID: *p.UnitId})
}
