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

var purchaseOrderLineLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.purchase_order_line")

// LoadPurchaseOrderLines fetches purchase order lines by ID, for a receiving or delivery line that
// names the line it was raised from. The backend scopes them through the order they belong to, so a
// line the caller cannot reach is absent from the result and the reference stays null.
func LoadPurchaseOrderLines(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}

	resp, apiErr := grpcutil.CallRPC(ctx, purchaseOrderLineLoaderTracer, "loader.purchase_order_lines.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetPurchaseOrderLinesByIDsResponse, error) {
			return corePurchaseClient.BatchGetPurchaseOrderLinesByIDs(ctx, &pb.BatchGetPurchaseOrderLinesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	// The received figure is computed, so it has no id and must carry its unit; resolve the units
	// the lines are counted in before presenting.
	unitIDs := make([]string, 0, len(resp.Lines))
	for _, l := range resp.Lines {
		unitIDs = append(unitIDs, l.QuantityUnitId)
	}
	units, apiErr := LoadUnitsByID(ctx, unitIDs...)
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Lines))
	for _, l := range resp.Lines {
		out[l.Id] = purchaseOrderLineFromProto(meta, l, units)
	}
	return out, nil
}

// The line's own quantity and price ride on it; the units they are counted in and the item it orders
// are stashed as ids, so a caller can reach through to them the same way they would from the order.
func purchaseOrderLineFromProto(meta *resourcekit.LoadMeta, info *pb.PurchaseOrderLineInfo, units map[string]*apiresource.Unit) *apiresource.PurchaseOrderLine {
	line := &apiresource.PurchaseOrderLine{
		ID:                 info.Id,
		Object:             constants.ObjectTypePurchaseOrderLine,
		LineItemNumber:     info.LineItemNumber,
		ProductSKU:         info.ProductSku,
		ProductDescription: info.ProductDescription,
		QuantityOrdered: &apiresource.Quantity{
			ID:           info.QuantityId,
			Object:       constants.ObjectTypeQuantity,
			Value:        info.QuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(info.QuantityValue, info.QuantityUnitAbbreviation, info.QuantityUnitType),
		},
		UnitPrice: &apiresource.Rate{
			ID:     info.UnitPriceId,
			Object: constants.ObjectTypeRate,
			Value:  info.UnitPriceValue,
			DisplayValue: apiresource.FormatRateDisplayValue(
				info.UnitPriceValue, info.UnitPriceNumeratorUnitAbbreviation, "", info.UnitPriceDenominatorUnitAbbreviation),
			CreatedAt: grpcutil.TimestampToTime(info.UnitPriceCreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(info.UnitPriceUpdatedAt),
		},
		CreatedAt: grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(info.UpdatedAt),
	}

	meta.Set(constants.ObjectTypeQuantity, info.QuantityId, "unit_id", info.QuantityUnitId)
	meta.Set(constants.ObjectTypeRate, info.UnitPriceId, "numerator_unit_id", info.UnitPriceNumeratorUnitId)
	meta.Set(constants.ObjectTypeRate, info.UnitPriceId, "denominator_unit_id", info.UnitPriceDenominatorUnitId)
	if info.ItemId != nil && *info.ItemId != "" {
		meta.Set(constants.ObjectTypePurchaseOrderLine, line.ID, "item_id", *info.ItemId)
	}

	// What has been received is rolled up from the receiving order rather than stored, so it is a
	// computed quantity: no id, and the unit travels with it.
	if info.QuantityReceivedValue != nil {
		line.QuantityReceived = &apiresource.ComputedQuantity{
			Object:       constants.ObjectTypeComputedQuantity,
			Value:        *info.QuantityReceivedValue,
			DisplayValue: apiresource.FormatDisplayValue(*info.QuantityReceivedValue, info.QuantityUnitAbbreviation, info.QuantityUnitType),
			Unit:         units[info.QuantityUnitId],
		}
	}

	return line
}
