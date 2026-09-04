package receivingorderep

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/ptrutil"
	"github.com/open-mrp/api/shared/safeconv"
	"github.com/open-mrp/api/shared/tracing"
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
	// CoreClient (required) is the core-service receiving gRPC client.
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
		Status:      req.Status.StringPtr(),
		ItemIds:     req.ItemIDs,
		SupplierIds: req.SupplierIDs,
		// Ask the backend to expand lines when requested (supplier/purchase_order
		// are resolved gateway-side).
		Includes: resourcekit.FilterIncludes(ctx, "lines"),
	}

	if req.StartDate != nil {
		t, err := grpcutil.ParseDateString(*req.StartDate)
		if err == nil {
			pbReq.StartDate = timestamppb.New(t)
		}
	}
	if req.EndDate != nil {
		t, err := grpcutil.ParseEndDateString(*req.EndDate)
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

	var lineInfos []*pb.ReceivingOrderLineInfo
	for _, o := range resp.ReceivingOrders {
		lineInfos = append(lineInfos, o.GetLines()...)
	}
	units, apiErr := resourceloaders.LoadUnitsByID(ctx, receivingOrderLineUnitIDs(lineInfos...)...)
	if apiErr != nil {
		return nil, apiErr
	}

	orders := make([]apiresource.ReceivingOrder, len(resp.ReceivingOrders))
	for i, o := range resp.ReceivingOrders {
		orders[i] = receivingOrderSummaryFromProto(ctx, o, units)
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

	units, apiErr := resourceloaders.LoadUnitsByID(ctx, receivingOrderLineUnitIDs(resp.ReceivingOrder.GetLines()...)...)
	if apiErr != nil {
		return nil, apiErr
	}

	result := receivingOrderFromProto(ctx, resp.ReceivingOrder, units)
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

	units, apiErr := resourceloaders.LoadUnitsByID(ctx, receivingOrderLineUnitIDs(resp.ReceivingOrder.GetLines()...)...)
	if apiErr != nil {
		return nil, apiErr
	}

	result := receivingOrderFromProto(ctx, resp.ReceivingOrder, units)
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

	units, apiErr := resourceloaders.LoadUnitsByID(ctx, receivingOrderLineUnitIDs(resp.ReceivingOrder.GetLines()...)...)
	if apiErr != nil {
		return nil, apiErr
	}

	result := receivingOrderFromProto(ctx, resp.ReceivingOrder, units)
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

	units, apiErr := resourceloaders.LoadUnitsByID(ctx, receivingOrderLineUnitIDs(resp.ReceivingOrder.GetLines()...)...)
	if apiErr != nil {
		return nil, apiErr
	}

	result := receivingOrderFromProto(ctx, resp.ReceivingOrder, units)
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

	units, apiErr := resourceloaders.LoadUnitsByID(ctx, receivingOrderLineUnitIDs(resp.Line)...)
	if apiErr != nil {
		return nil, apiErr
	}

	result := receivingOrderLineFromProto(resp.Line, units)
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

	units, apiErr := resourceloaders.LoadUnitsByID(ctx, receivingOrderLineUnitIDs(resp.Line)...)
	if apiErr != nil {
		return nil, apiErr
	}

	result := receivingOrderLineFromProto(resp.Line, units)
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

	units, apiErr := resourceloaders.LoadUnitsByID(ctx, receivingOrderLineUnitIDs(resp.Line)...)
	if apiErr != nil {
		return nil, apiErr
	}

	result := receivingOrderLineFromProto(resp.Line, units)
	return &result, nil
}

// receivingOrderSummaryFromProto maps a list-view ReceivingOrderSummaryInfo to
// the merged ReceivingOrder. Expandable references (purchase_order, supplier)
// are left nil and populated via the include resolver from stashed FK ids.
func receivingOrderSummaryFromProto(ctx context.Context, info *pb.ReceivingOrderSummaryInfo, units map[string]*apiresource.Unit) apiresource.ReceivingOrder {
	if info == nil {
		return apiresource.ReceivingOrder{}
	}

	r := apiresource.ReceivingOrder{
		ID:          info.Id,
		Object:      constants.ObjectTypeReceivingOrder,
		Number:      info.Number,
		LineCount:   info.LineCount,
		CompletedAt: grpcutil.TimestampToTimePtr(info.CompletedAt),
		CreatedAt:   grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:   grpcutil.TimestampToTime(info.UpdatedAt),
	}

	stashReceivingOrderFKs(ctx, info.Id, info.SupplierId, info.SupplierName, info.SupplierNumber, info.PurchaseOrderId, info.PurchaseOrderNumber, info.PurchaseOrderStatus, info.Totals, info.Deliveries)

	// Lines are populated on the summary only when the list includes them.
	if len(info.Lines) > 0 {
		meta := resourcekit.GetLoadMeta(ctx)
		lines := make([]apiresource.ReceivingOrderLine, len(info.Lines))
		for i, l := range info.Lines {
			lines[i] = receivingOrderLineFromProto(l, units)
			stashReceivingOrderLineMeta(meta, l, &lines[i])
		}
		meta.Set(constants.ObjectTypeReceivingOrder, info.Id, "lines",
			apiresource.NewList(lines, apiresource.PageInfo{}))
	}

	return r
}

// receivingOrderFromProto maps a detail ReceivingOrderInfo to the merged
// ReceivingOrder. Expandable references (purchase_order, supplier, lines) are
// left nil and populated via the include resolver from stashed meta.
func receivingOrderFromProto(ctx context.Context, info *pb.ReceivingOrderInfo, units map[string]*apiresource.Unit) apiresource.ReceivingOrder {
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

	stashReceivingOrderFKs(ctx, info.Id, info.SupplierId, info.SupplierName, info.SupplierNumber, info.PurchaseOrderId, info.PurchaseOrderNumber, info.PurchaseOrderStatus, info.Totals, info.Deliveries)

	// Lines (expandable): stash the pre-built list plus each line's order_line
	// reference so the include resolver can populate them on ?include=lines and
	// ?include=lines.order_line.
	if len(info.Lines) > 0 {
		meta := resourcekit.GetLoadMeta(ctx)
		lines := make([]apiresource.ReceivingOrderLine, len(info.Lines))
		for i, l := range info.Lines {
			lines[i] = receivingOrderLineFromProto(l, units)
			stashReceivingOrderLineMeta(meta, l, &lines[i])
		}
		meta.Set(constants.ObjectTypeReceivingOrder, info.Id, "lines",
			apiresource.NewList(lines, apiresource.PageInfo{}))
	}

	return r
}

// stashReceivingOrderFKs stashes the sub-objects the include resolver reveals on request.
//
// The supplier is the seller account — cross-account, so not resolvable through the account-scoped loader — and totals and related are computed with the order rather than fetched, so all three are carried inline from what the query already returned. Never fabricate the referenced documents.
func stashReceivingOrderFKs(ctx context.Context, id string, supplierID, supplierName, supplierNumber *string, purchaseOrderID, purchaseOrderNumber, purchaseOrderStatus string, totals *pb.ReceivingOrderTotalsInfo, deliveries []*pb.DocumentRefInfo) {
	meta := resourcekit.GetLoadMeta(ctx)
	if supplierID != nil {
		meta.Set(constants.ObjectTypeReceivingOrder, id, "supplier", &apiresource.Supplier{
			ID:     *supplierID,
			Object: constants.ObjectTypeSupplier,
			Name:   ptrutil.Deref(supplierName),
			Number: ptrutil.Deref(supplierNumber),
		})
	}
	related := &apiresource.ReceivingOrderRelated{Object: constants.ObjectTypeReceivingOrderRelated}
	if purchaseOrderID != "" {
		po := apiresource.NewRecord(purchaseOrderID, constants.RecordTypePurchaseOrder)
		if purchaseOrderNumber != "" {
			po.Number = &purchaseOrderNumber
		}
		if purchaseOrderStatus != "" {
			po.Status = &purchaseOrderStatus
		}
		related.PurchaseOrder = po
	}
	if recs := documentRecords(deliveries, constants.RecordTypeDelivery); recs != nil {
		related.Deliveries = recs
	}
	if related.PurchaseOrder != nil || related.Deliveries != nil {
		meta.Set(constants.ObjectTypeReceivingOrder, id, "related", related)
	}
	if t := receivingOrderTotalsFromProto(totals); t != nil {
		meta.Set(constants.ObjectTypeReceivingOrder, id, "totals", t)
	}
}

// receivingOrderTotalsFromProto turns the aggregated amounts into the resource's totals, dividing each stage's amount by the ordered amount to get its completion.
//
// Completion is a ratio of amounts rather than of quantities because a receiving order's lines can each count in a different unit; summing those quantities would add pairs to metres, and the ratio would be meaningless.
//
// An order whose lines total nothing ordered has no meaningful completion, so the stages report zero rather than dividing by it.
func receivingOrderTotalsFromProto(info *pb.ReceivingOrderTotalsInfo) *apiresource.ReceivingOrderTotals {
	if info == nil {
		return nil
	}

	ordered := decimalOrZero(info.OrderedAmount)
	stage := func(amount string) apiresource.ReceivingOrderStageTotal {
		total := apiresource.ReceivingOrderStageTotal{
			Object: constants.ObjectTypeReceivingOrderStageTotal,
			Amount: amountOrZero(amount),
		}
		if ordered.IsPositive() {
			completion, _ := decimalOrZero(amount).Div(ordered).Float64()
			total.Completion = completion
		}
		return total
	}

	return &apiresource.ReceivingOrderTotals{
		Object:   constants.ObjectTypeReceivingOrderTotals,
		Ordered:  amountOrZero(info.OrderedAmount),
		Stocked:  stage(info.StockedAmount),
		Rejected: stage(info.RejectedAmount),
	}
}

func decimalOrZero(v string) decimal.Decimal {
	d, err := decimal.NewFromString(v)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func amountOrZero(v string) string {
	if v == "" {
		return "0"
	}
	return v
}

func receivingOrderLineFromProto(info *pb.ReceivingOrderLineInfo, units map[string]*apiresource.Unit) apiresource.ReceivingOrderLine {
	if info == nil {
		return apiresource.ReceivingOrderLine{}
	}

	line := apiresource.ReceivingOrderLine{
		ID:             info.Id,
		Object:         constants.ObjectTypeReceivingOrderLine,
		LineItemNumber: info.OrderLineItemNumber,
		Quantity: &apiresource.Quantity{
			ID:           info.QuantityId,
			Object:       constants.ObjectTypeQuantity,
			Value:        info.QuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(info.QuantityValue, info.QuantityUnitAbbreviation, ""),
			// Unit left nil: expandable, loaded with real data via `lines.quantity.unit`; never fabricated.
		},
		StockedAt: grpcutil.TimestampToTimePtr(info.StockedAt),
		CreatedAt: grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(info.UpdatedAt),
	}

	// What was refused is summed across the line's deliveries rather than stored, so it is a
	// computed quantity — no id, and it carries the unit it was summed in.
	if info.RejectedQuantityValue != nil {
		line.RejectedQuantity = &apiresource.ComputedQuantity{
			Object:       constants.ObjectTypeComputedQuantity,
			Value:        *info.RejectedQuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(*info.RejectedQuantityValue, info.QuantityUnitAbbreviation, ""),
			Unit:         units[info.QuantityUnitId],
		}
	}

	return line
}

// receivingOrderLineUnitIDs names the units a set of lines is measured in, so a caller can resolve them before presenting.
func receivingOrderLineUnitIDs(lines ...*pb.ReceivingOrderLineInfo) []string {
	ids := make([]string, 0, len(lines)*2)
	for _, l := range lines {
		if l == nil {
			continue
		}
		ids = append(ids, l.QuantityUnitId, l.OrderLineUnitId)
	}
	return ids
}

// stashReceivingOrderLineMeta stashes the sub-objects a line reveals on request: the item it receives, and the quantity the purchase order asked for.
func stashReceivingOrderLineMeta(meta *resourcekit.LoadMeta, info *pb.ReceivingOrderLineInfo, line *apiresource.ReceivingOrderLine) {
	if info == nil || line.Quantity == nil {
		return
	}
	if info.OrderLineItemId != nil && *info.OrderLineItemId != "" {
		meta.Set(constants.ObjectTypeReceivingOrderLine, line.ID, "item_id", *info.OrderLineItemId)
	}
	// The purchase order line this line receives against, resolved through the PO line loader.
	meta.Set(constants.ObjectTypeReceivingOrderLine, line.ID, "order_line_id", info.OrderLineId)
	// The unit the line's own quantity is counted in, for `lines.quantity.unit`.
	meta.Set(constants.ObjectTypeQuantity, line.Quantity.ID, "unit_id", info.QuantityUnitId)

	// What the purchase order asked for is the order line's own quantity, reported here under that
	// quantity's id rather than as a copy of its value.
	meta.Set(constants.ObjectTypeReceivingOrderLine, line.ID, "quantity_ordered", &apiresource.Quantity{
		ID:           info.OrderLineQuantityId,
		Object:       constants.ObjectTypeQuantity,
		Value:        info.OrderLineQuantityOrdered,
		DisplayValue: apiresource.FormatDisplayValue(info.OrderLineQuantityOrdered, info.OrderLineUnitAbbreviation, ""),
	})
	meta.Set(constants.ObjectTypeQuantity, info.OrderLineQuantityId, "unit_id", info.OrderLineUnitId)
}

// documentRecords turns the cross-links a purchasing document carries into record references. The ids, numbers and statuses are what the query already returned, so nothing here is invented.
func documentRecords(refs []*pb.DocumentRefInfo, recordType constants.RecordType) *apiresource.List[apiresource.Record] {
	if len(refs) == 0 {
		return nil
	}
	records := make([]apiresource.Record, len(refs))
	for i, r := range refs {
		rec := apiresource.NewRecord(r.Id, recordType)
		if r.Number != "" {
			number := r.Number
			rec.Number = &number
		}
		if r.Status != "" {
			status := r.Status
			rec.Status = &status
		}
		records[i] = *rec
	}
	return apiresource.NewList(records, apiresource.PageInfo{})
}
