package grpc

import (
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/patch"
	pb "github.com/augno/api/shared/proto/core"
)

func quantityPatchToDomain(p *pb.QuantityPatch) patch.Field[domain.QuantityInput] {
	if p == nil {
		return patch.Unset[domain.QuantityInput]()
	}
	if p.Clear {
		return patch.Clear[domain.QuantityInput]()
	}
	if p.Value == nil || p.UnitId == nil {
		return patch.Clear[domain.QuantityInput]()
	}
	return patch.Set(domain.QuantityInput{Value: *p.Value, UnitID: *p.UnitId})
}

func stringListPatchToSliceField(p *pb.StringListPatch) patch.Field[[]string] {
	f := patch.StringListFieldFromProto(p)
	if f.IsUnset() {
		return patch.Unset[[]string]()
	}
	if f.IsClear() {
		return patch.Clear[[]string]()
	}
	sl, _ := f.Value()
	return patch.Set([]string(sl))
}
