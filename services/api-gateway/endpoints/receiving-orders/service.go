package receivingorderep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/safeconv"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var receivingOrderSvcTracer = tracing.GetTracer("api-gateway.endpoints.receiving-orders.service")

type ReceivingOrderSvc interface {
	ListReceivingOrders(ctx context.Context, req *ListReceivingOrdersRequest) (*apiresource.List[apiresource.ReceivingOrder], *apierror.APIError)
	GetReceivingOrder(ctx context.Context, req *RetrieveReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError)
	StockReceivingOrder(ctx context.Context, req *StockReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError)
	ReceiveReceivingOrder(ctx context.Context, req *ReceiveReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError)
	VoidReceivingOrder(ctx context.Context, req *VoidReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError)
	UpdateReceivingOrderLine(ctx context.Context, req *UpdateReceivingOrderLineRequest) (*apiresource.ReceivingOrderLine, *apierror.APIError)
	VoidReceivingOrderLine(ctx context.Context, req *VoidReceivingOrderLineRequest) (*apiresource.ReceivingOrderLine, *apierror.APIError)
	ReceiveReceivingOrderLine(ctx context.Context, req *ReceiveReceivingOrderLineRequest) (*apiresource.ReceivingOrderLine, *apierror.APIError)
}

type ReceivingOrderSvcConfig struct {
	CoreClient pb.CoreReceivingServiceClient
}

func (c *ReceivingOrderSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("receiving order endpoint service: core client is required")
	}
	return nil
}

type receivingOrderSvcImpl struct {
	coreClient pb.CoreReceivingServiceClient
}

func NewReceivingOrderSvc(config *ReceivingOrderSvcConfig) ReceivingOrderSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &receivingOrderSvcImpl{coreClient: config.CoreClient}
}

func (m *receivingOrderSvcImpl) ListReceivingOrders(ctx context.Context, req *ListReceivingOrdersRequest) (*apiresource.List[apiresource.ReceivingOrder], *apierror.APIError) {
	pbReq := &pb.ListReceivingOrdersRequest{
		Cursor:      req.Cursor,
		Limit:       req.Limit,
		Query:       req.Query,
		Status:      req.Status,
		ItemIds:     req.ItemIDs,
		SupplierIds: req.SupplierIDs,
	}

	if req.StartDate != nil {
		t, err := grpcutil.ParseDateString(*req.StartDate)
		if err == nil {
			pbReq.StartDate = timestamppb.New(t)
		}
	}
	if req.EndDate != nil {
		t, err := grpcutil.ParseDateString(*req.EndDate)
		if err == nil {
			pbReq.EndDate = timestamppb.New(t)
		}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, receivingOrderSvcTracer, "service.receiving_orders.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListReceivingOrdersResponse, error) {
			return m.coreClient.ListReceivingOrders(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	if resp == nil {
		return apiresource.NewList[apiresource.ReceivingOrder](nil, apiresource.PageInfo{}), nil
	}

	orders := make([]apiresource.ReceivingOrder, len(resp.ReceivingOrders))
	for i, o := range resp.ReceivingOrders {
		orders[i] = receivingOrderSummaryFromProto(ctx, o)
	}

	return apiresource.NewList(orders, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *receivingOrderSvcImpl) GetReceivingOrder(ctx context.Context, req *RetrieveReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError) {
	pbReq := &pb.GetReceivingOrderRequest{
		Id: req.ReceivingOrderID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, receivingOrderSvcTracer, "service.receiving_orders.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetReceivingOrderResponse, error) {
			return m.coreClient.GetReceivingOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := receivingOrderFromProto(ctx, resp.ReceivingOrder)
	return &result, nil
}

func (m *receivingOrderSvcImpl) StockReceivingOrder(ctx context.Context, req *StockReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError) {
	lineItems := make([]*pb.StockingLineItemInfo, len(req.LineItems))
	for i, li := range req.LineItems {
		allocations := make([]*pb.StorageAllocationInfo, len(li.Allocations))
		for j, a := range li.Allocations {
			allocations[j] = &pb.StorageAllocationInfo{
				LocationId: a.LocationID.Ptr(),
				Quantity:   a.Quantity,
			}
		}
		lineItems[i] = &pb.StockingLineItemInfo{
			ReceivingOrderLineId: li.ReceivingOrderLineID,
			LotNumber:            li.LotNumber.Ptr(),
			RejectedQuantity:     li.RejectedQuantity.Ptr(),
			Allocations:          allocations,
		}
	}

	pbReq := &pb.StockReceivingOrderRequest{
		Id: req.ReceivingOrderID,
		Data: &pb.StockingDataInfo{
			LineItems: lineItems,
		},
	}

	resp, apiErr := grpcutil.CallRPC(ctx, receivingOrderSvcTracer, "service.receiving_orders.stock", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.StockReceivingOrderResponse, error) {
			return m.coreClient.StockReceivingOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := receivingOrderFromProto(ctx, resp.ReceivingOrder)
	return &result, nil
}

func (m *receivingOrderSvcImpl) ReceiveReceivingOrder(ctx context.Context, req *ReceiveReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError) {
	pbReq := &pb.ReceiveReceivingOrderRequest{Id: req.ReceivingOrderID}

	resp, apiErr := grpcutil.CallRPC(ctx, receivingOrderSvcTracer, "service.receiving_orders.receive", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ReceiveReceivingOrderResponse, error) {
			return m.coreClient.ReceiveReceivingOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := receivingOrderFromProto(ctx, resp.ReceivingOrder)
	return &result, nil
}

func (m *receivingOrderSvcImpl) VoidReceivingOrder(ctx context.Context, req *VoidReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError) {
	pbReq := &pb.VoidReceivingOrderRequest{Id: req.ReceivingOrderID}

	resp, apiErr := grpcutil.CallRPC(ctx, receivingOrderSvcTracer, "service.receiving_orders.void", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.VoidReceivingOrderResponse, error) {
			return m.coreClient.VoidReceivingOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := receivingOrderFromProto(ctx, resp.ReceivingOrder)
	return &result, nil
}

func (m *receivingOrderSvcImpl) UpdateReceivingOrderLine(ctx context.Context, req *UpdateReceivingOrderLineRequest) (*apiresource.ReceivingOrderLine, *apierror.APIError) {
	pbReq := &pb.UpdateReceivingOrderLineRequest{
		ReceivingOrderId: req.ReceivingOrderID,
		Id:               req.LineID,
	}
	if v, ok := req.QuantityValue.Value(); ok {
		pbReq.QuantityValue = &v
	}

	resp, apiErr := grpcutil.CallRPC(ctx, receivingOrderSvcTracer, "service.receiving_orders.update_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateReceivingOrderLineResponse, error) {
			return m.coreClient.UpdateReceivingOrderLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := receivingOrderLineFromProto(resp.Line)
	return &result, nil
}

func (m *receivingOrderSvcImpl) VoidReceivingOrderLine(ctx context.Context, req *VoidReceivingOrderLineRequest) (*apiresource.ReceivingOrderLine, *apierror.APIError) {
	pbReq := &pb.VoidReceivingOrderLineRequest{
		ReceivingOrderId: req.ReceivingOrderID,
		Id:               req.LineID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, receivingOrderSvcTracer, "service.receiving_orders.void_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.VoidReceivingOrderLineResponse, error) {
			return m.coreClient.VoidReceivingOrderLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := receivingOrderLineFromProto(resp.Line)
	return &result, nil
}

func (m *receivingOrderSvcImpl) ReceiveReceivingOrderLine(ctx context.Context, req *ReceiveReceivingOrderLineRequest) (*apiresource.ReceivingOrderLine, *apierror.APIError) {
	pbReq := &pb.ReceiveReceivingOrderLineRequest{
		ReceivingOrderId: req.ReceivingOrderID,
		Id:               req.LineID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, receivingOrderSvcTracer, "service.receiving_orders.receive_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ReceiveReceivingOrderLineResponse, error) {
			return m.coreClient.ReceiveReceivingOrderLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := receivingOrderLineFromProto(resp.Line)
	return &result, nil
}

// receivingOrderSummaryFromProto maps a list-view ReceivingOrderSummaryInfo to
// the merged ReceivingOrder. Expandable references (purchase_order, supplier)
// are left nil and populated via the include resolver from stashed FK ids.
func receivingOrderSummaryFromProto(ctx context.Context, info *pb.ReceivingOrderSummaryInfo) apiresource.ReceivingOrder {
	if info == nil {
		return apiresource.ReceivingOrder{}
	}

	r := apiresource.ReceivingOrder{
		ID:                   info.Id,
		Object:               constants.ObjectTypeReceivingOrder,
		Number:               info.Number,
		LineCount:            info.LineCount,
		CompletionPercentage: info.CompletionPercentage,
		CompletedAt:          grpcutil.TimestampToTimePtr(info.CompletedAt),
		CreatedAt:            grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:            grpcutil.TimestampToTime(info.UpdatedAt),
	}

	stashReceivingOrderFKs(ctx, info.Id, info.SupplierId, info.PurchaseOrderId)

	return r
}

// receivingOrderFromProto maps a detail ReceivingOrderInfo to the merged
// ReceivingOrder. Expandable references (purchase_order, supplier, lines) are
// left nil and populated via the include resolver from stashed meta.
func receivingOrderFromProto(ctx context.Context, info *pb.ReceivingOrderInfo) apiresource.ReceivingOrder {
	if info == nil {
		return apiresource.ReceivingOrder{}
	}

	r := apiresource.ReceivingOrder{
		ID:          info.Id,
		Object:      constants.ObjectTypeReceivingOrder,
		Number:      info.Number,
		Note:        info.Note,
		LineCount:   safeconv.IntToInt32(len(info.Lines)),
		CompletedAt: grpcutil.TimestampToTimePtr(info.CompletedAt),
		CreatedAt:   grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:   grpcutil.TimestampToTime(info.UpdatedAt),
	}

	stashReceivingOrderFKs(ctx, info.Id, info.SupplierId, info.PurchaseOrderId)

	// Lines (expandable): stash the pre-built list plus each line's order_line
	// reference so the include resolver can populate them on ?include=lines and
	// ?include=lines.order_line.
	if len(info.Lines) > 0 {
		meta := resourcekit.GetLoadMeta(ctx)
		lines := make([]apiresource.ReceivingOrderLine, len(info.Lines))
		for i, l := range info.Lines {
			lines[i] = receivingOrderLineFromProto(l)
			stashReceivingOrderLineMeta(meta, l, &lines[i])
		}
		meta.Set(constants.ObjectTypeReceivingOrder, info.Id, "lines",
			apiresource.NewList(lines, apiresource.PageInfo{}))
	}

	return r
}

// stashReceivingOrderFKs stashes the supplier and purchase_order FK ids so the
// loader-backed include resolver fetches the real resources on ?include=.
// Never fabricate the referenced documents.
func stashReceivingOrderFKs(ctx context.Context, id string, supplierID *string, purchaseOrderID string) {
	meta := resourcekit.GetLoadMeta(ctx)
	if supplierID != nil {
		meta.Set(constants.ObjectTypeReceivingOrder, id, "supplier_id", *supplierID)
	}
	if purchaseOrderID != "" {
		meta.Set(constants.ObjectTypeReceivingOrder, id, "purchase_order_id", purchaseOrderID)
	}
}

func receivingOrderLineFromProto(info *pb.ReceivingOrderLineInfo) apiresource.ReceivingOrderLine {
	if info == nil {
		return apiresource.ReceivingOrderLine{}
	}

	ts := grpcutil.TimestampToTime(info.CreatedAt)

	line := apiresource.ReceivingOrderLine{
		ID:     info.Id,
		Object: constants.ObjectTypeReceivingOrderLine,
		Quantity: &apiresource.Quantity{
			ID:           info.QuantityId,
			Object:       constants.ObjectTypeQuantity,
			Value:        info.QuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(info.QuantityValue, info.QuantityUnitAbbreviation, ""),
			// Unit left nil: expandable, loaded with real data via ?include=; never fabricated.
		},
		// OrderLine is expandable: left nil here, populated from stashed meta on
		// ?include=lines.order_line.
		StockedAt: grpcutil.TimestampToTimePtr(info.StockedAt),
		CreatedAt: ts,
		UpdatedAt: grpcutil.TimestampToTime(info.UpdatedAt),
	}

	if info.RejectedQuantityValue != nil {
		line.RejectedQuantity = &apiresource.Quantity{
			ID:           info.Id + "_rejected",
			Object:       constants.ObjectTypeQuantity,
			Value:        *info.RejectedQuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(*info.RejectedQuantityValue, info.QuantityUnitAbbreviation, ""),
			// Unit left nil: expandable, loaded with real data via ?include=; never fabricated.
		}
	}

	return line
}

// buildOrderLineForReceivingLine builds a pre-built, new-shape SalesOrderLine
// reference from the receiving line's identifying proto fields. There is no
// standalone sales-order-line loader, so the parent proto carries the line's
// identifying fields. Only the required base fields are set; the expandable
// money/quantity fields (Product, QuantityOrdered, UnitPrice, UnitCost, Totals)
// are left nil and are never fabricated.
func buildOrderLineForReceivingLine(info *pb.ReceivingOrderLineInfo) *apiresource.SalesOrderLine {
	ts := grpcutil.TimestampToTime(info.CreatedAt)
	sku := "—"
	if info.OrderLineItemSku != nil && *info.OrderLineItemSku != "" {
		sku = *info.OrderLineItemSku
	}
	var productDesc *string
	if info.OrderLineItemDescription != nil && *info.OrderLineItemDescription != "" {
		productDesc = info.OrderLineItemDescription
	}

	return &apiresource.SalesOrderLine{
		ID:                 info.OrderLineId,
		Object:             constants.ObjectTypeSalesOrderLine,
		LineItemNumber:     1,
		ProductSKU:         sku,
		ProductDescription: productDesc,
		CreatedAt:          ts,
		UpdatedAt:          ts,
	}
}

// stashReceivingOrderLineMeta stashes a line's expandable order_line reference
// so the include resolver can populate it on ?include=lines.order_line.
func stashReceivingOrderLineMeta(meta *resourcekit.LoadMeta, info *pb.ReceivingOrderLineInfo, line *apiresource.ReceivingOrderLine) {
	meta.Set(constants.ObjectTypeReceivingOrderLine, line.ID, "order_line",
		buildOrderLineForReceivingLine(info))
}
