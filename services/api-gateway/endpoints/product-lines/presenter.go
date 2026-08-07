package productlineep

import (
	"strconv"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	pb "github.com/augno/api/shared/proto/core"
)

func ProductLinePresenter(pl *pb.ProductLineInfo, ownerAccount *apiresource.Account) apiresource.ProductLine {
	if pl == nil {
		return apiresource.ProductLine{}
	}

	result := apiresource.ProductLine{
		ID:               pl.Id,
		Object:           constants.ObjectTypeProductLine,
		Name:             pl.Name,
		Description:      pl.Description,
		Notes:            pl.Notes,
		CommissionPolicy: constants.CommissionPolicy(pl.CommissionPolicy),
		FreightPolicy:    constants.FreightPolicy(pl.FreightPolicy),
		Owner:            apiresource.NewOwnerWithAccount(pl.AccountId, ownerAccount),
		CreatedAt:        grpcutil.TimestampToTime(pl.CreatedAt),
		UpdatedAt:        grpcutil.TimestampToTime(pl.UpdatedAt),
	}

	if pl.FulfillmentPolicyCode != nil && *pl.FulfillmentPolicyCode != "" {
		policy := constants.FulfillmentPolicy(*pl.FulfillmentPolicyCode)
		result.FulfillmentPolicy = &policy
	}

	if pl.UnitGroup != nil {
		ug := &apiresource.UnitGroup{
			ID:        pl.UnitGroup.Id,
			Object:    constants.ObjectTypeUnitGroup,
			Name:      pl.UnitGroup.Name,
			Type:      constants.UnitType(pl.UnitGroup.Type),
			CreatedAt: grpcutil.TimestampToTime(pl.UnitGroup.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(pl.UnitGroup.UpdatedAt),
		}
		if pl.UnitGroup.BaseUnit != nil {
			u := pl.UnitGroup.BaseUnit
			mapped := apiresource.Unit{
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
				CreatedAt:         grpcutil.TimestampToTime(u.CreatedAt),
				UpdatedAt:         grpcutil.TimestampToTime(u.UpdatedAt),
			}
			ug.BaseUnit = &mapped
		}
		if len(pl.UnitGroup.AssociatedUnits) > 0 {
			units := make([]apiresource.UnitGroupUnit, 0, len(pl.UnitGroup.AssociatedUnits))
			for _, au := range pl.UnitGroup.AssociatedUnits {
				if au == nil {
					continue
				}
				discountPct, _ := strconv.ParseFloat(au.DiscountPercentage, 64)
				discountFixed, _ := strconv.ParseFloat(au.DiscountFixed, 64)
				visibility := constants.CustomerPortalVisibilityHidden
				if au.IsVisible {
					visibility = constants.CustomerPortalVisibilityVisible
				}
				ugu := apiresource.UnitGroupUnit{
					ID:                       au.Id,
					Object:                   constants.ObjectTypeUnitGroupUnit,
					DiscountPercentage:       discountPct,
					DiscountFixed:            discountFixed,
					CustomerPortalVisibility: visibility,
					CreatedAt:                grpcutil.TimestampToTime(au.CreatedAt),
					UpdatedAt:                grpcutil.TimestampToTime(au.UpdatedAt),
				}
				if au.Unit != nil {
					u := au.Unit
					mapped := apiresource.Unit{
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
						CreatedAt:         grpcutil.TimestampToTime(u.CreatedAt),
						UpdatedAt:         grpcutil.TimestampToTime(u.UpdatedAt),
					}
					ugu.Unit = &mapped
				}
				units = append(units, ugu)
			}
			ug.AssociatedUnits = apiresource.NewList(units, apiresource.PageInfo{})
		}
		result.UnitGroup = ug
	}

	return result
}
