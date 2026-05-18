package itemcategoryep

import (
	"context"
	"strconv"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	pb "github.com/augno/api/shared/proto/core"
)

func ItemCategoryPresenter(ic *pb.ItemCategoryInfo, ownerAccount *apiresource.Account) apiresource.ItemCategory {
	if ic == nil {
		return apiresource.ItemCategory{}
	}

	result := apiresource.ItemCategory{
		ID:        ic.Id,
		Object:    constants.ObjectTypeItemCategory,
		Name:      ic.Name,
		Notes:     ic.Notes,
		Type:      constants.ItemCategoryType(ic.ItemCategoryTypeCode),
		Owner:     apiresource.NewOwnerWithAccount(ic.AccountId, ownerAccount),
		CreatedAt: grpcutil.TimestampToTime(ic.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(ic.UpdatedAt),
	}

	if len(ic.Properties) > 0 {
		props := make([]apiresource.Property, len(ic.Properties))
		for i, p := range ic.Properties {
			props[i] = apiresource.Property{
				ID:        p.Id,
				Object:    constants.ObjectTypeProperty,
				Name:      p.Name,
				CreatedAt: grpcutil.TimestampToTime(p.CreatedAt),
				UpdatedAt: grpcutil.TimestampToTime(p.UpdatedAt),
			}
		}
		result.Properties = apiresource.NewList(props, apiresource.PageInfo{})
	}

	if ic.UnitGroup != nil {
		ug := &apiresource.UnitGroup{
			ID:        ic.UnitGroup.Id,
			Object:    constants.ObjectTypeUnitGroup,
			Name:      ic.UnitGroup.Name,
			Type:      constants.UnitType(ic.UnitGroup.Type),
			CreatedAt: grpcutil.TimestampToTime(ic.UnitGroup.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(ic.UnitGroup.UpdatedAt),
		}
		if ic.UnitGroup.BaseUnit != nil {
			u := ic.UnitGroup.BaseUnit
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
		if len(ic.UnitGroup.AssociatedUnits) > 0 {
			units := make([]apiresource.UnitGroupUnit, 0, len(ic.UnitGroup.AssociatedUnits))
			for _, au := range ic.UnitGroup.AssociatedUnits {
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

func ItemCategoryListPresenter(ctx context.Context, resp *pb.ListItemCategoriesResponse, ownerAccount *apiresource.Account) *apiresource.List[apiresource.ItemCategory] {
	if resp == nil {
		return apiresource.NewList[apiresource.ItemCategory](nil, apiresource.PageInfo{})
	}

	categories := make([]apiresource.ItemCategory, len(resp.ItemCategories))
	for i, ic := range resp.ItemCategories {
		categories[i] = ItemCategoryPresenter(ic, ownerAccount)
	}

	return apiresource.NewList(categories, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
