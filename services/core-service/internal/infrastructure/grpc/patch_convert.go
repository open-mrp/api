package grpc

import (
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/field"
	pb "github.com/open-mrp/api/shared/proto/core"
)

func quantityPatchToDomain(p *pb.QuantityPatch) field.Clearable[domain.QuantityInput] {
	if p == nil {
		return field.Unset[domain.QuantityInput]()
	}
	if p.Clear {
		return field.Clear[domain.QuantityInput]()
	}
	if p.Value == nil || p.UnitId == nil {
		return field.Clear[domain.QuantityInput]()
	}
	return field.Set(domain.QuantityInput{Value: *p.Value, UnitID: *p.UnitId})
}

func stringListPatchToSliceField(p *pb.StringListPatch) field.Clearable[[]string] {
	f := field.StringListClearableFromProto(p)
	if f.IsUnset() {
		return field.Unset[[]string]()
	}
	if f.IsClear() {
		return field.Clear[[]string]()
	}
	sl, _ := f.Value()
	return field.Set([]string(sl))
}
