package partep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func lightRatePresenter(r *pb.RateInfo) *apiresource.Rate {
	if r == nil {
		return nil
	}
	return &apiresource.Rate{
		ID:     r.Id,
		Object: constants.ObjectTypeRate,
		Value:  r.Value,
		NumeratorUnit: &apiresource.Unit{
			ID:     r.NumeratorUnitId,
			Object: constants.ObjectTypeUnit,
		},
		DenominatorUnit: &apiresource.Unit{
			ID:     r.DenominatorUnitId,
			Object: constants.ObjectTypeUnit,
		},
		DisplayValue: "",
	}
}

func lightItemCategoryPresenter(c *pb.ItemCategoryInfo) *apiresource.ItemCategory {
	if c == nil {
		return nil
	}
	return &apiresource.ItemCategory{
		ID:     c.Id,
		Object: constants.ObjectTypeItemCategory,
		Name:   c.Name,
		Owner:  apiresource.NewOwner(c.AccountId),
	}
}

func lightAttributePresenter(a *pb.ItemAttributeInfo) *apiresource.Attribute {
	if a == nil {
		return nil
	}
	return &apiresource.Attribute{
		ID:        a.Id,
		Object:    constants.ObjectTypeAttribute,
		Value:     a.Value,
		ColorCode: constants.Color(a.GetColorCode()),
		SortOrder: a.SortOrder,
	}
}

func PartPresenter(p *pb.PartInfo) apiresource.Part {
	if p == nil || p.Item == nil {
		return apiresource.Part{}
	}

	i := p.Item
	part := apiresource.Part{
		ID:          i.Id,
		Object:      constants.ObjectTypePart,
		SKU:         i.Sku,
		Description: i.Description,
		Notes:       i.Notes,
		Category:    lightItemCategoryPresenter(i.Category),
		UnitValue:   lightRatePresenter(i.UnitValue),
		UnitCost:    lightRatePresenter(i.UnitCost),
		BurnRate:    lightRatePresenter(i.BurnRate),
		IsDirty:     i.IsDirty,
		CreatedAt:   grpcutil.TimestampToTime(i.CreatedAt),
		UpdatedAt:   grpcutil.TimestampToTime(i.UpdatedAt),
	}

	if i.Attributes != nil {
		attrs := make([]apiresource.Attribute, len(i.Attributes))
		for j, a := range i.Attributes {
			if ap := lightAttributePresenter(a); ap != nil {
				attrs[j] = *ap
			}
		}
		part.Attributes = apiresource.NewList(attrs, apiresource.PageInfo{})
	}

	return part
}

func PartListPresenter(resp *pb.ListPartsResponse) *apiresource.List[apiresource.Part] {
	if resp == nil {
		return apiresource.NewList[apiresource.Part](nil, apiresource.PageInfo{})
	}

	parts := make([]apiresource.Part, len(resp.Parts))
	for i, part := range resp.Parts {
		parts[i] = PartPresenter(part)
	}

	return apiresource.NewList(parts, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
