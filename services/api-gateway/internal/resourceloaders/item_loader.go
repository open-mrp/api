package resourceloaders

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

var itemLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.item")

func LoadItems(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, itemLoaderTracer, "loader.items.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetItemsByIDsResponse, error) {
			return coreClient.BatchGetItemsByIDs(ctx, &pb.BatchGetItemsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Items))

	propertyIDs := map[string]struct{}{}
	for _, item := range resp.Items {
		for _, a := range item.Attributes {
			if a != nil && a.PropertyId != "" {
				propertyIDs[a.PropertyId] = struct{}{}
			}
		}
	}
	propertyMap := map[string]*apiresource.Property{}
	if len(propertyIDs) > 0 {
		ids := make([]string, 0, len(propertyIDs))
		for id := range propertyIDs {
			ids = append(ids, id)
		}
		// Returned rather than swallowed: presenting every attribute's property as null on a
		// transient lookup failure is indistinguishable from a property that was deleted, so the
		// caller cannot tell it should retry.
		loaded, apiErr := LoadProperties(ctx, ids)
		if apiErr != nil {
			return nil, apiErr
		}
		for id, v := range loaded {
			propertyMap[id] = v.(*apiresource.Property)
		}
	}

	for _, item := range resp.Items {
		out[item.Id] = itemFromProto(item)

		if item.Category != nil {
			meta.Set(constants.ObjectTypeItem, item.Id, "item_category_id", item.Category.Id)
		}

		if item.UnitValue != nil {
			meta.Set(constants.ObjectTypeItem, item.Id, "unit_value", rateFromProto(item.UnitValue))
		}
		if item.UnitCost != nil {
			meta.Set(constants.ObjectTypeItem, item.Id, "unit_cost", rateFromProto(item.UnitCost))
		}
		if item.BurnRate != nil {
			meta.Set(constants.ObjectTypeItem, item.Id, "burn_rate", rateFromProto(item.BurnRate))
		}

		attrs := make([]apiresource.Attribute, 0, len(item.Attributes))
		for _, a := range item.Attributes {
			if a == nil {
				continue
			}
			attr := apiresource.Attribute{
				ID:        a.Id,
				Object:    constants.ObjectTypeAttribute,
				Value:     a.Value,
				SortOrder: a.SortOrder,
				CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
				UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
			}
			if a.ColorCode != nil {
				attr.ColorCode = constants.Color(*a.ColorCode)
			}
			// Left nil when the property did not resolve: an attribute naming a property whose
			// record is missing says so by omitting it, not by inventing one with a blank name.
			if a.PropertyId != "" {
				attr.Property = propertyMap[a.PropertyId]
			}
			attrs = append(attrs, attr)
		}
		meta.Set(constants.ObjectTypeItem, item.Id, "attributes_list",
			apiresource.NewList(attrs, apiresource.PageInfo{}))
	}
	return out, nil
}

func itemFromProto(i *pb.ItemInfo) *apiresource.Item {
	return &apiresource.Item{
		ID:           i.Id,
		Object:       constants.ObjectTypeItem,
		SKU:          i.Sku,
		Description:  i.Description,
		Notes:        i.Notes,
		ItemTypeCode: constants.ItemTypeCode(i.ItemTypeCode),
		CreatedAt:    grpcutil.TimestampToTime(i.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(i.UpdatedAt),
	}
}

func rateFromProto(r *pb.RateInfo) *apiresource.Rate {
	normalizedValue := apiresource.NormalizeRateValue(r.Value)
	return &apiresource.Rate{
		ID:     r.Id,
		Object: constants.ObjectTypeRate,
		Value:  normalizedValue,
		NumeratorUnit: &apiresource.Unit{
			ID:                r.NumeratorUnitId,
			Object:            constants.ObjectTypeUnit,
			Name:              r.NumeratorUnitName,
			Abbreviation:      r.NumeratorUnitAbbreviation,
			Type:              constants.UnitType(r.NumeratorUnitType),
			RatioNumerator:    r.NumeratorUnitRatioNumerator,
			RatioDenominator:  r.NumeratorUnitRatioDenominator,
			OffsetNumerator:   r.NumeratorUnitOffsetNumerator,
			OffsetDenominator: r.NumeratorUnitOffsetDenominator,
			CreatedAt:         grpcutil.TimestampToTime(r.NumeratorUnitCreatedAt),
			UpdatedAt:         grpcutil.TimestampToTime(r.NumeratorUnitUpdatedAt),
		},
		DenominatorUnit: &apiresource.Unit{
			ID:                r.DenominatorUnitId,
			Object:            constants.ObjectTypeUnit,
			Name:              r.DenominatorUnitName,
			Abbreviation:      r.DenominatorUnitAbbreviation,
			Type:              constants.UnitType(r.DenominatorUnitType),
			RatioNumerator:    r.DenominatorUnitRatioNumerator,
			RatioDenominator:  r.DenominatorUnitRatioDenominator,
			OffsetNumerator:   r.DenominatorUnitOffsetNumerator,
			OffsetDenominator: r.DenominatorUnitOffsetDenominator,
			CreatedAt:         grpcutil.TimestampToTime(r.DenominatorUnitCreatedAt),
			UpdatedAt:         grpcutil.TimestampToTime(r.DenominatorUnitUpdatedAt),
		},
		DisplayValue: apiresource.FormatRateDisplayValue(
			normalizedValue,
			r.NumeratorUnitAbbreviation,
			r.NumeratorUnitType,
			r.DenominatorUnitAbbreviation,
		),
		CreatedAt: grpcutil.TimestampToTime(r.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(r.UpdatedAt),
	}
}
