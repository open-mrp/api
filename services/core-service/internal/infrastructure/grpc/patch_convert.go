package grpc

import (
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/fieldpb"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/apikit/field"
)

func stringListPatchToSliceField(p *pb.StringListPatch) field.Clearable[[]string] {
	f := fieldpb.StringListClearableFromProto(p)
	if f.IsUnset() {
		return field.Unset[[]string]()
	}
	if f.IsClear() {
		return field.Clear[[]string]()
	}
	sl, _ := f.Value()
	return field.Set([]string(sl))
}

// inlineAddressToDomain is nil when the request sent no inline address.
func inlineAddressToDomain(p *pb.InlineAddressInput) *domain.InlineAddressParams {
	if p == nil {
		return nil
	}
	return &domain.InlineAddressParams{
		ID:                p.Id,
		Name:              p.Name,
		Phone:             fieldpb.StringClearableFromProto(p.Phone),
		Email:             fieldpb.StringClearableFromProto(p.Email),
		IsDropShip:        p.IsDropShip,
		ReceiveCalendarID: fieldpb.StringClearableFromProto(p.ReceiveCalendarId),
		StreetLine1:       p.StreetLine_1,
		StreetLine2:       fieldpb.StringClearableFromProto(p.StreetLine_2),
		Locality:          p.Locality,
		State:             p.State,
		PostalCode:        p.PostalCode,
		Country:           p.Country,
	}
}
