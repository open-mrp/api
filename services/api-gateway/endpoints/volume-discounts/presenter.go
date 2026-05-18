package volumediscountep

import (
	"context"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func VolumeDiscountPresenter(d *pb.VolumeDiscountInfo) apiresource.VolumeDiscount {
	if d == nil {
		return apiresource.VolumeDiscount{}
	}

	tiers := make([]apiresource.VolumeDiscountTier, len(d.Tiers))
	for i, t := range d.Tiers {
		tiers[i] = apiresource.VolumeDiscountTier{
			ID:                 t.Id,
			Object:             constants.ObjectTypeVolumeDiscountTier,
			Name:               t.Name,
			DiscountPercentage: t.DiscountPercentage,
			Threshold:          t.Threshold,
			CreatedAt:          grpcutil.TimestampToTime(t.CreatedAt),
			UpdatedAt:          grpcutil.TimestampToTime(t.UpdatedAt),
		}
	}

	customerGroups := make([]apiresource.AccountGroup, len(d.CustomerGroups))
	for i, cg := range d.CustomerGroups {
		customerGroups[i] = apiresource.AccountGroup{
			ID:        cg.AccountGroupId,
			Object:    constants.ObjectTypeAccountGroup,
			Name:      cg.Name,
			CreatedAt: grpcutil.TimestampToTime(cg.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(cg.UpdatedAt),
		}
		if cg.CommissionPolicy != nil {
			customerGroups[i].CommissionPolicy = constants.CommissionPolicy(*cg.CommissionPolicy)
		}
		if cg.FreightPolicy != nil {
			customerGroups[i].FreightPolicy = constants.FreightPolicy(*cg.FreightPolicy)
		}
		if cg.Type != nil {
			customerGroups[i].Type = constants.AccountGroupType(*cg.Type)
		}
	}

	productLines := make([]apiresource.ProductLine, len(d.ProductLines))
	for i, pl := range d.ProductLines {
		productLines[i] = apiresource.ProductLine{
			ID:        pl.Id,
			Object:    constants.ObjectTypeProductLine,
			Name:      pl.Name,
			CreatedAt: grpcutil.TimestampToTime(pl.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(pl.UpdatedAt),
		}
		if pl.CommissionPolicy != nil {
			productLines[i].CommissionPolicy = constants.CommissionPolicy(*pl.CommissionPolicy)
		}
		if pl.FreightPolicy != nil {
			productLines[i].FreightPolicy = constants.FreightPolicy(*pl.FreightPolicy)
		}
	}

	categories := make([]apiresource.ItemCategory, len(d.Categories))
	for i, cat := range d.Categories {
		categories[i] = apiresource.ItemCategory{
			ID:        cat.Id,
			Object:    constants.ObjectTypeItemCategory,
			Name:      cat.Name,
			CreatedAt: grpcutil.TimestampToTime(cat.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(cat.UpdatedAt),
		}
		if cat.Type != nil {
			categories[i].Type = constants.ItemCategoryType(*cat.Type)
		}
	}

	attributes := make([]apiresource.Attribute, len(d.Attributes))
	for i, attr := range d.Attributes {
		attributes[i] = apiresource.Attribute{
			ID:        attr.Id,
			Object:    constants.ObjectTypeAttribute,
			Value:     attr.Name,
			CreatedAt: grpcutil.TimestampToTime(attr.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(attr.UpdatedAt),
		}
		if attr.ColorCode != nil {
			attributes[i].ColorCode = constants.Color(*attr.ColorCode)
		}
	}

	units := make([]apiresource.Unit, len(d.AcceptableUnits))
	for i, u := range d.AcceptableUnits {
		units[i] = apiresource.Unit{
			ID:                u.Id,
			Object:            constants.ObjectTypeUnit,
			Name:              u.Name,
			Abbreviation:      u.Abbreviation,
			Type:              constants.UnitType(u.Type),
			RatioNumerator:    u.RatioNumerator,
			RatioDenominator:  u.RatioDenominator,
			OffsetNumerator:   u.OffsetNumerator,
			OffsetDenominator: u.OffsetDenominator,
			CreatedAt:         grpcutil.TimestampToTime(u.CreatedAt),
			UpdatedAt:         grpcutil.TimestampToTime(u.UpdatedAt),
		}
	}

	return apiresource.VolumeDiscount{
		ID:              d.Id,
		Object:          constants.ObjectTypeVolumeDiscount,
		Name:            d.Name,
		Tiers:           apiresource.NewList(tiers, apiresource.PageInfo{}),
		CustomerGroups:  apiresource.NewList(customerGroups, apiresource.PageInfo{}),
		ProductLines:    apiresource.NewList(productLines, apiresource.PageInfo{}),
		Categories:      apiresource.NewList(categories, apiresource.PageInfo{}),
		Attributes:      apiresource.NewList(attributes, apiresource.PageInfo{}),
		AcceptableUnits: apiresource.NewList(units, apiresource.PageInfo{}),
		CreatedAt:       grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:       grpcutil.TimestampToTime(d.UpdatedAt),
	}
}

func VolumeDiscountListPresenter(ctx context.Context, resp *pb.ListVolumeDiscountsResponse) *apiresource.List[apiresource.VolumeDiscount] {
	if resp == nil {
		return apiresource.NewList[apiresource.VolumeDiscount](nil, apiresource.PageInfo{})
	}

	discounts := make([]apiresource.VolumeDiscount, len(resp.VolumeDiscounts))
	for i, d := range resp.VolumeDiscounts {
		discounts[i] = VolumeDiscountPresenter(d)
	}

	return apiresource.NewList(discounts, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
