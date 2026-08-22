package resourceloaders

import (
	"context"
	"strconv"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

var productLineLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.product_line")

// LoadProductLines fetches product lines by ID via BatchGetProductLinesByIDs. Stashes owner_account_id and the pre-built UnitGroup in LoadMeta for SubField closures.
func LoadProductLines(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, productLineLoaderTracer, "loader.product_lines.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetProductLinesByIDsResponse, error) {
			return coreClient.BatchGetProductLinesByIDs(ctx, &pb.BatchGetProductLinesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.ProductLines))
	for _, pl := range resp.ProductLines {
		out[pl.Id] = productLineFromProto(pl)

		var accountID string
		if pl.AccountId != nil {
			accountID = *pl.AccountId
		}
		meta.Set(constants.ObjectTypeProductLine, pl.Id, "owner_account_id", accountID)

		// The lot is a Quantity built inline and stashed for the default_lot SubField, the same way a customer's credit_limit is. This loader backs both retrieve and list, so a lot mapped only in the presenter would read as null everywhere.
		if pl.DefaultLot != nil {
			meta.Set(constants.ObjectTypeProductLine, pl.Id, "default_lot", buildProductLineDefaultLot(pl.DefaultLot))
			if pl.DefaultLot.UnitId != "" {
				meta.Set(constants.ObjectTypeQuantity, pl.DefaultLot.Id, "unit_id", pl.DefaultLot.UnitId)
			}
		}

		// Pre-build the UnitGroup from the proto and stash it for the unit_group SubField's Populate. This preserves backward compat: requesting ?include[]=unit_group returns the full UnitGroup with BaseUnit and AssociatedUnits inline.
		if pl.UnitGroup != nil {
			ug := buildUnitGroupFromProto(pl.UnitGroup)
			meta.Set(constants.ObjectTypeProductLine, pl.Id, "unit_group", ug)
			stashUnitGroupMeta(meta, ug)
		}
	}
	return out, nil
}

// buildProductLineDefaultLot renders the lot as a Quantity. Unit is left nil: the unit id is stashed so LoadUnits fetches the real Unit on ?include=default_lot.unit rather than this fabricating one.
func buildProductLineDefaultLot(lot *pb.ProductLineDefaultLotInfo) *apiresource.Quantity {
	return &apiresource.Quantity{
		ID:           lot.Id,
		Object:       constants.ObjectTypeQuantity,
		Value:        apiresource.NormalizeQuantityValue(lot.Value, ""),
		DisplayValue: apiresource.FormatDisplayValue(lot.Value, "", ""),
	}
}

func productLineFromProto(pl *pb.ProductLineInfo) *apiresource.ProductLine {
	out := &apiresource.ProductLine{
		ID:               pl.Id,
		Object:           constants.ObjectTypeProductLine,
		Name:             pl.Name,
		Description:      pl.Description,
		Notes:            pl.Notes,
		CommissionPolicy: constants.CommissionPolicy(pl.CommissionPolicy),
		FreightPolicy:    constants.FreightPolicy(pl.FreightPolicy),
		CreatedAt:        grpcutil.TimestampToTime(pl.CreatedAt),
		UpdatedAt:        grpcutil.TimestampToTime(pl.UpdatedAt),
	}
	if pl.FulfillmentPolicyCode != nil && *pl.FulfillmentPolicyCode != "" {
		policy := constants.FulfillmentPolicy(*pl.FulfillmentPolicyCode)
		out.FulfillmentPolicy = &policy
	}
	return out
}

func stashUnitGroupMeta(meta *resourcekit.LoadMeta, ug *apiresource.UnitGroup) {
	if ug.BaseUnit != nil {
		meta.Set(constants.ObjectTypeUnitGroup, ug.ID, "base_unit_id", ug.BaseUnit.ID)
	}
	if ug.AssociatedUnits != nil {
		meta.Set(constants.ObjectTypeUnitGroup, ug.ID, "associated_units_data", ug.AssociatedUnits.Data)
		for i := range ug.AssociatedUnits.Data {
			if ug.AssociatedUnits.Data[i].Unit != nil {
				meta.Set(constants.ObjectTypeUnitGroupUnit, ug.AssociatedUnits.Data[i].ID, "unit_id", ug.AssociatedUnits.Data[i].Unit.ID)
			}
		}
	}
}

func buildUnitGroupFromProto(ug *pb.ItemCategoryUnitGroupInfo) *apiresource.UnitGroup {
	result := &apiresource.UnitGroup{
		ID:        ug.Id,
		Object:    constants.ObjectTypeUnitGroup,
		Name:      ug.Name,
		Type:      constants.UnitType(ug.Type),
		CreatedAt: grpcutil.TimestampToTime(ug.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(ug.UpdatedAt),
	}
	if ug.BaseUnit != nil {
		u := ug.BaseUnit
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
		result.BaseUnit = &mapped
	}
	if len(ug.AssociatedUnits) > 0 {
		items := make([]apiresource.UnitGroupUnit, 0, len(ug.AssociatedUnits))
		for _, au := range ug.AssociatedUnits {
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
			items = append(items, ugu)
		}
		result.AssociatedUnits = apiresource.NewList(items, apiresource.PageInfo{})
	}
	return result
}
