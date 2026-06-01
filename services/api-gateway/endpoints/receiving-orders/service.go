package receivingorderep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var receivingOrderSvcTracer = tracing.GetTracer("api-gateway.endpoints.receiving-orders.service")

type ReceivingOrderSvc interface {
	ListReceivingOrders(ctx context.Context, req *ListReceivingOrdersRequest) (*apiresource.List[apiresource.ReceivingOrderSummary], *apierror.APIError)
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

func (m *receivingOrderSvcImpl) ListReceivingOrders(ctx context.Context, req *ListReceivingOrdersRequest) (*apiresource.List[apiresource.ReceivingOrderSummary], *apierror.APIError) {
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

	return receivingOrderListFromProto(ctx, resp), nil
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

	result := receivingOrderFromProto(resp.ReceivingOrder)
	return &result, nil
}

func (m *receivingOrderSvcImpl) StockReceivingOrder(ctx context.Context, req *StockReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError) {
	lineItems := make([]*pb.StockingLineItemInfo, len(req.LineItems))
	for i, li := range req.LineItems {
		allocations := make([]*pb.StorageAllocationInfo, len(li.Allocations))
		for j, a := range li.Allocations {
			allocations[j] = &pb.StorageAllocationInfo{
				LocationId: a.LocationID,
				Quantity:   a.Quantity,
			}
		}
		lineItems[i] = &pb.StockingLineItemInfo{
			ReceivingOrderLineId: li.ReceivingOrderLineID,
			LotNumber:            li.LotNumber,
			RejectedQuantity:     li.RejectedQuantity,
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

	result := receivingOrderFromProto(resp.ReceivingOrder)
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

	result := receivingOrderFromProto(resp.ReceivingOrder)
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

	result := receivingOrderFromProto(resp.ReceivingOrder)
	return &result, nil
}

func (m *receivingOrderSvcImpl) UpdateReceivingOrderLine(ctx context.Context, req *UpdateReceivingOrderLineRequest) (*apiresource.ReceivingOrderLine, *apierror.APIError) {
	pbReq := &pb.UpdateReceivingOrderLineRequest{
		ReceivingOrderId: req.ReceivingOrderID,
		Id:               req.LineID,
	}
	if req.QuantityValue != nil {
		pbReq.QuantityValue = req.QuantityValue
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

func receivingOrderSummaryFromProto(info *pb.ReceivingOrderSummaryInfo) apiresource.ReceivingOrderSummary {
	if info == nil {
		return apiresource.ReceivingOrderSummary{}
	}

	ts := grpcutil.TimestampToTime(info.CreatedAt)

	s := apiresource.ReceivingOrderSummary{
		ID:                   info.Id,
		Object:               constants.ObjectTypeReceivingOrder,
		Number:               info.Number,
		PurchaseOrder:        apiresource.ExpandableSalesOrderStub(info.PurchaseOrderId, info.PurchaseOrderNumber, ts),
		LineCount:            info.LineCount,
		CompletionPercentage: info.CompletionPercentage,
		CompletedAt:          grpcutil.TimestampToTimePtr(info.CompletedAt),
		CreatedAt:            ts,
		UpdatedAt:            grpcutil.TimestampToTime(info.UpdatedAt),
	}

	if info.SupplierId != nil {
		supplier := apiresource.ExpandableAccountStub(*info.SupplierId, "", ts)
		if info.SupplierName != nil {
			supplier.Name = *info.SupplierName
		}
		s.Supplier = supplier
	}

	return s
}

func receivingOrderFromProto(info *pb.ReceivingOrderInfo) apiresource.ReceivingOrder {
	if info == nil {
		return apiresource.ReceivingOrder{}
	}

	ts := grpcutil.TimestampToTime(info.CreatedAt)

	r := apiresource.ReceivingOrder{
		ID:            info.Id,
		Object:        constants.ObjectTypeReceivingOrder,
		Number:        info.Number,
		PurchaseOrder: apiresource.ExpandableSalesOrderStub(info.PurchaseOrderId, info.PurchaseOrderNumber, ts),
		CompletedAt:   grpcutil.TimestampToTimePtr(info.CompletedAt),
		CreatedAt:     ts,
		UpdatedAt:     grpcutil.TimestampToTime(info.UpdatedAt),
	}

	if info.SupplierId != nil {
		supplier := apiresource.ExpandableAccountStub(*info.SupplierId, "", ts)
		if info.SupplierName != nil {
			supplier.Name = *info.SupplierName
		}
		r.Supplier = supplier
	}

	if info.Note != nil {
		note := *info.Note
		r.Note = &note
	}

	if len(info.Lines) > 0 {
		lines := make([]apiresource.ReceivingOrderLine, len(info.Lines))
		for i, l := range info.Lines {
			lines[i] = receivingOrderLineFromProto(l)
		}
		r.Lines = apiresource.NewList(lines, apiresource.PageInfo{})
	}

	return r
}

func receivingOrderLineFromProto(info *pb.ReceivingOrderLineInfo) apiresource.ReceivingOrderLine {
	if info == nil {
		return apiresource.ReceivingOrderLine{}
	}

	ts := grpcutil.TimestampToTime(info.CreatedAt)
	orderLineSKU := "ITEM"
	if info.OrderLineItemSku != nil && *info.OrderLineItemSku != "" {
		orderLineSKU = *info.OrderLineItemSku
	}

	line := apiresource.ReceivingOrderLine{
		ID:     info.Id,
		Object: constants.ObjectTypeReceivingOrderLine,
		Quantity: &apiresource.Quantity{
			ID:           info.QuantityId,
			Object:       constants.ObjectTypeQuantity,
			Value:        info.QuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(info.QuantityValue, info.QuantityUnitAbbreviation, ""),
			Unit:         apiresource.ExpandableUnitStub(info.QuantityUnitId, "", info.QuantityUnitAbbreviation, string(constants.UnitTypeQuantity), ts),
		},
		OrderLine: &apiresource.SalesOrderLineDetail{
			ID:             info.OrderLineId,
			Object:         constants.ObjectTypeSalesOrderLine,
			LineItemNumber: 1,
			ProductSKU:     orderLineSKU,
			QuantityOrdered: &apiresource.Quantity{
				ID:           info.OrderLineId + "_ordered",
				Object:       constants.ObjectTypeQuantity,
				Value:        info.OrderLineQuantityOrdered,
				DisplayValue: apiresource.FormatDisplayValue(info.OrderLineQuantityOrdered, info.OrderLineUnitAbbreviation, ""),
				Unit:         apiresource.ExpandableUnitStub(info.OrderLineUnitId, "", info.OrderLineUnitAbbreviation, string(constants.UnitTypeQuantity), ts),
			},
			UnitPrice: &apiresource.Rate{
				ID:              info.OrderLineId + "_unit_price",
				Object:          constants.ObjectTypeRate,
				Value:           "0",
				NumeratorUnit:   apiresource.ExpandableUnitStub("dollar", "US Dollar", "$", string(constants.UnitTypeCurrency), ts),
				DenominatorUnit: apiresource.ExpandableUnitStub(info.OrderLineUnitId, "", info.OrderLineUnitAbbreviation, string(constants.UnitTypeQuantity), ts),
				DisplayValue:    apiresource.FormatRateDisplayValue("0", "$", string(constants.UnitTypeCurrency), info.OrderLineUnitAbbreviation),
				CreatedAt:       ts,
				UpdatedAt:       ts,
			},
			CreatedAt: ts,
			UpdatedAt: ts,
		},
		StockedAt: grpcutil.TimestampToTimePtr(info.StockedAt),
		CreatedAt: ts,
		UpdatedAt: grpcutil.TimestampToTime(info.UpdatedAt),
	}

	if info.OrderLineItemId != nil {
		item := apiresource.ExpandableItemStub(*info.OrderLineItemId, orderLineSKU, ts)
		line.OrderLine.Item = item
	}

	if info.OrderLineItemDescription != nil {
		line.OrderLine.ProductDescription = info.OrderLineItemDescription
	}

	if info.OrderLineItemSku != nil {
		line.OrderLine.ProductSKU = *info.OrderLineItemSku
	}

	if info.RejectedQuantityValue != nil {
		line.RejectedQuantity = &apiresource.Quantity{
			ID:           info.Id + "_rejected",
			Object:       constants.ObjectTypeQuantity,
			Value:        *info.RejectedQuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(*info.RejectedQuantityValue, info.QuantityUnitAbbreviation, ""),
			Unit:         apiresource.ExpandableUnitStub(info.QuantityUnitId, "", info.QuantityUnitAbbreviation, string(constants.UnitTypeQuantity), ts),
		}
	}

	return line
}

func receivingOrderListFromProto(ctx context.Context, resp *pb.ListReceivingOrdersResponse) *apiresource.List[apiresource.ReceivingOrderSummary] {
	if resp == nil {
		return apiresource.NewList[apiresource.ReceivingOrderSummary](nil, apiresource.PageInfo{})
	}

	orders := make([]apiresource.ReceivingOrderSummary, len(resp.ReceivingOrders))
	for i, o := range resp.ReceivingOrders {
		orders[i] = receivingOrderSummaryFromProto(o)
	}

	return apiresource.NewList(orders, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
