package unitgroupep

import (
	"strconv"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	pb "github.com/augno/api/shared/proto/core"
)

func UnitGroupPresenter(ug *pb.UnitGroupInfo) apiresource.UnitGroup {
	if ug == nil {
		return apiresource.UnitGroup{}
	}

	var baseUnit *apiresource.Unit
	if ug.BaseUnit != nil {
		u := apiresource.Unit{
			ID:                ug.BaseUnit.Id,
			Object:            constants.ObjectTypeUnit,
			Name:              ug.BaseUnit.Name,
			Abbreviation:      ug.BaseUnit.Abbreviation,
			Type:              constants.UnitType(ug.BaseUnit.Type),
			RatioNumerator:    db.TrimDecimal(ug.BaseUnit.RatioNumerator),
			RatioDenominator:  db.TrimDecimal(ug.BaseUnit.RatioDenominator),
			OffsetNumerator:   db.TrimDecimal(ug.BaseUnit.OffsetNumerator),
			OffsetDenominator: db.TrimDecimal(ug.BaseUnit.OffsetDenominator),
			IsBaseUnit:        ug.BaseUnit.IsBaseUnit,
			Owner:             apiresource.NewOwner(ug.BaseUnit.AccountId),
			CreatedAt:         grpcutil.TimestampToTime(ug.BaseUnit.CreatedAt),
			UpdatedAt:         grpcutil.TimestampToTime(ug.BaseUnit.UpdatedAt),
		}
		baseUnit = &u
	}

	var associatedUnits *apiresource.List[apiresource.UnitGroupUnit]
	if ug.UnitConversions != nil {
		items := make([]apiresource.UnitGroupUnit, len(ug.UnitConversions))
		for i, c := range ug.UnitConversions {
			items[i] = UnitGroupUnitPresenter(c)
		}
		associatedUnits = apiresource.NewList(items, apiresource.PageInfo{})
	}

	return apiresource.UnitGroup{
		ID:              ug.Id,
		Object:          constants.ObjectTypeUnitGroup,
		Name:            ug.Name,
		Notes:           ug.Notes,
		Type:            constants.UnitType(ug.Type),
		BaseUnit:        baseUnit,
		AssociatedUnits: associatedUnits,
		Owner:           apiresource.NewOwner(ug.AccountId),
		CreatedAt:       grpcutil.TimestampToTime(ug.CreatedAt),
		UpdatedAt:       grpcutil.TimestampToTime(ug.UpdatedAt),
	}
}

func UnitGroupUnitPresenter(u *pb.UnitGroupUnitInfo) apiresource.UnitGroupUnit {
	if u == nil {
		return apiresource.UnitGroupUnit{}
	}

	discountPercentage, _ := strconv.ParseFloat(u.DiscountPercentage, 64)
	discountFixed, _ := strconv.ParseFloat(u.DiscountFixed, 64)

	visibility := constants.CustomerPortalVisibilityHidden
	if u.IsVisible {
		visibility = constants.CustomerPortalVisibilityVisible
	}

	var unit *apiresource.Unit
	if u.Unit != nil {
		mapped := apiresource.Unit{
			ID:                u.Unit.Id,
			Object:            constants.ObjectTypeUnit,
			Name:              u.Unit.Name,
			Abbreviation:      u.Unit.Abbreviation,
			Type:              constants.UnitType(u.Unit.Type),
			RatioNumerator:    db.TrimDecimal(u.Unit.RatioNumerator),
			RatioDenominator:  db.TrimDecimal(u.Unit.RatioDenominator),
			OffsetNumerator:   db.TrimDecimal(u.Unit.OffsetNumerator),
			OffsetDenominator: db.TrimDecimal(u.Unit.OffsetDenominator),
			IsBaseUnit:        u.Unit.IsBaseUnit,
			Owner:             apiresource.NewOwner(u.Unit.AccountId),
			CreatedAt:         grpcutil.TimestampToTime(u.Unit.CreatedAt),
			UpdatedAt:         grpcutil.TimestampToTime(u.Unit.UpdatedAt),
		}
		unit = &mapped
	}

	return apiresource.UnitGroupUnit{
		ID:                       u.Id,
		Object:                   constants.ObjectTypeUnitGroupUnit,
		Unit:                     unit,
		DiscountPercentage:       discountPercentage,
		DiscountFixed:            discountFixed,
		CustomerPortalVisibility: visibility,
		CreatedAt:                grpcutil.TimestampToTime(u.CreatedAt),
		UpdatedAt:                grpcutil.TimestampToTime(u.UpdatedAt),
	}
}

func UnitGroupUnitListPresenter(resp *pb.ListUnitGroupUnitsResponse) *apiresource.List[apiresource.UnitGroupUnit] {
	if resp == nil {
		return apiresource.NewList[apiresource.UnitGroupUnit](nil, apiresource.PageInfo{})
	}

	units := make([]apiresource.UnitGroupUnit, len(resp.UnitGroupUnits))
	for i, u := range resp.UnitGroupUnits {
		units[i] = UnitGroupUnitPresenter(u)
	}

	return apiresource.NewList(units, apiresource.PageInfo{})
}

func UnitGroupListPresenter(resp *pb.ListUnitGroupsResponse) *apiresource.List[apiresource.UnitGroup] {
	if resp == nil {
		return apiresource.NewList[apiresource.UnitGroup](nil, apiresource.PageInfo{})
	}

	unitGroups := make([]apiresource.UnitGroup, len(resp.UnitGroups))
	for i, ug := range resp.UnitGroups {
		unitGroups[i] = UnitGroupPresenter(ug)
	}

	return apiresource.NewList(unitGroups, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
