package unitep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func UnitPresenter(u *pb.UnitInfo) apiresource.Unit {
	if u == nil {
		return apiresource.Unit{}
	}

	return apiresource.Unit{
		ID:                u.Id,
		Object:            constants.ObjectTypeUnit,
		Name:              u.Name,
		Abbreviation:      u.Abbreviation,
		Type:              constants.UnitType(u.Type),
		RatioNumerator:    u.RatioNumerator,
		RatioDenominator:  u.RatioDenominator,
		OffsetNumerator:   u.OffsetNumerator,
		OffsetDenominator: u.OffsetDenominator,
		IsBaseUnit:        u.IsBaseUnit,
		IsInternal:        u.IsInternal,
		CreatedAt:         grpcutil.TimestampToTime(u.CreatedAt),
		UpdatedAt:         grpcutil.TimestampToTime(u.UpdatedAt),
	}
}

func UnitListPresenter(resp *pb.ListUnitsResponse) *apiresource.List[apiresource.Unit] {
	if resp == nil {
		return apiresource.NewList[apiresource.Unit](nil, apiresource.PageInfo{})
	}

	units := make([]apiresource.Unit, len(resp.Units))
	for i, u := range resp.Units {
		units[i] = UnitPresenter(u)
	}

	return apiresource.NewList(units, mapProtoPageInfo(resp.PageInfo))
}

func mapProtoPageInfo(pi *pb.PageInfo) apiresource.PageInfo {
	if pi == nil {
		return apiresource.PageInfo{}
	}
	return apiresource.PageInfo{
		NextCursor:  pi.NextCursor,
		PrevCursor:  pi.PrevCursor,
		HasNextPage: pi.HasNextPage,
		HasPrevPage: pi.HasPrevPage,
	}
}
