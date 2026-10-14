package resourceloaders

import (
	"context"
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
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

	meta := include.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Lines))
	for _, l := range resp.Lines {
		out[l.Id] = purchaseOrderLineFromProto(meta, l)
	}
	return out, nil
}

// The line's own quantity and price ride on it; the item it orders is stashed as an id, so a caller can
// reach through to it the same way they would from the order.
func purchaseOrderLineFromProto(meta *include.LoadMeta, info *pb.PurchaseOrderLineInfo) *apiresource.PurchaseOrderLine {
	line := &apiresource.PurchaseOrderLine{
		ID:                 info.Id,
		Object:             constants.ObjectTypePurchaseOrderLine,
		LineItemNumber:     info.LineItemNumber,
		ProductSKU:         info.ProductSku,
		ProductDescription: info.ProductDescription,
		QuantityOrdered:    apiresource.NewQuantity(info.QuantityValue, info.QuantityUnitId),
		UnitPrice:          UnitPriceFromProto(info.UnitPrice),
		CreatedAt:          grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:          grpcutil.TimestampToTime(info.UpdatedAt),
	}

	if info.ItemId != nil && *info.ItemId != "" {
		meta.Set(constants.ObjectTypePurchaseOrderLine, line.ID, "item_id", *info.ItemId)
	}

	// What has been received is rolled up from the receiving order rather than stored.
	if info.QuantityReceivedValue != nil {
		line.QuantityReceived = apiresource.NewQuantity(*info.QuantityReceivedValue, info.QuantityUnitId)
	}

	return line
}
