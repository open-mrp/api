package unitep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
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
		RatioNumerator:    db.TrimDecimal(u.RatioNumerator),
		RatioDenominator:  db.TrimDecimal(u.RatioDenominator),
		OffsetNumerator:   db.TrimDecimal(u.OffsetNumerator),
		OffsetDenominator: db.TrimDecimal(u.OffsetDenominator),
		IsBaseUnit:        u.IsBaseUnit,
		Owner:             apiresource.NewOwner(u.AccountId),
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

	return apiresource.NewList(units, grpcutil.MapProtoPageInfo(resp.PageInfo))
}

func ValidateUnitsPresenter(resp *pb.ValidateUnitsResponse) *apiresource.ValidateUnitsResponse {
	if resp == nil {
		return &apiresource.ValidateUnitsResponse{
			Object: constants.ObjectTypeMap,
			Units:  map[string]*apiresource.Unit{},
		}
	}

	units := make(map[string]*apiresource.Unit, len(resp.Units))
	for key, proto := range resp.Units {
		u := UnitPresenter(proto)
		units[key] = &u
	}

	return &apiresource.ValidateUnitsResponse{
		Object: constants.ObjectTypeMap,
		Units:  units,
	}
}
