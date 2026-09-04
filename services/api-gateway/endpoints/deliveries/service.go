package deliveryep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DeliverySvc interface {
	ListDeliveries(ctx context.Context, req *ListDeliveriesRequest) (*apiresource.List[apiresource.Delivery], *apierror.APIError)
	GetDelivery(ctx context.Context, req *RetrieveDeliveryRequest) (*apiresource.Delivery, *apierror.APIError)
}

type DeliverySvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type deliverySvcImpl struct {
	coreClient pb.CoreServiceClient
}

var deliverySvcTracer = tracing.GetTracer("api-gateway.endpoints.deliveries.service")

func (c *DeliverySvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("delivery endpoint service: core client is required")
	}
	return nil
}

func NewDeliverySvc(config *DeliverySvcConfig) DeliverySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &deliverySvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *deliverySvcImpl) ListDeliveries(ctx context.Context, req *ListDeliveriesRequest) (*apiresource.List[apiresource.Delivery], *apierror.APIError) {
	pbReq := &pb.ListDeliveriesRequest{
		Cursor:      req.Cursor,
		Limit:       req.Limit,
		Query:       req.Query,
		Status:      req.Status.StringPtr(),
		ItemIds:     req.ItemIDs,
		SupplierIds: req.SupplierIDs,
		// Only lines cost the backend anything; the related purchase order is carried on the row the query already returns.
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

	resp, apiErr := grpcutil.CallRPC(ctx, deliverySvcTracer, "service.deliveries.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListDeliveriesResponse, error) {
			return m.coreClient.ListDeliveries(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return deliveryListFromProto(ctx, resp), nil
}

func (m *deliverySvcImpl) GetDelivery(ctx context.Context, req *RetrieveDeliveryRequest) (*apiresource.Delivery, *apierror.APIError) {
	pbReq := &pb.GetDeliveryRequest{
		Id: req.DeliveryID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, deliverySvcTracer, "service.deliveries.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetDeliveryResponse, error) {
			return m.coreClient.GetDelivery(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := deliveryFromProto(resp.Delivery)
	stashDeliveryMeta(ctx, &result, resp.Delivery)
	return &result, nil
}

// deliverySummaryFromProto builds the base Delivery from the list-shaped proto. Expandable sub-objects are left nil and stashed by stashDeliverySummaryMeta, so they appear only when asked for.
func deliverySummaryFromProto(d *pb.DeliverySummaryInfo) apiresource.Delivery {
	if d == nil {
		return apiresource.Delivery{}
	}

	return apiresource.Delivery{
		ID:         d.Id,
		Object:     constants.ObjectTypeDelivery,
		Number:     d.Number,
		Status:     constants.DeliveryStatus(d.Status),
		AcceptedAt: grpcutil.TimestampToTimePtr(d.AcceptedAt),
		RejectedAt: grpcutil.TimestampToTimePtr(d.RejectedAt),
		CreatedAt:  grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(d.UpdatedAt),
	}
}

func stashDeliverySummaryMeta(ctx context.Context, d *apiresource.Delivery, info *pb.DeliverySummaryInfo) {
	if info == nil {
		return
	}
	stashDeliveryRelated(ctx, d.ID, info.PurchaseOrderId, info.PurchaseOrderNumber, info.PurchaseOrderStatus, info.ReceivingOrderId, info.ReceivingOrderNumber, info.ReceivingOrderStatus)
	// Lines are populated on the summary only when the list request includes them.
	if len(info.Lines) > 0 {
		lines := make([]apiresource.DeliveryLine, len(info.Lines))
		for i, l := range info.Lines {
			lines[i] = deliveryLineFromProto(l)
			stashDeliveryLineMeta(resourcekit.GetLoadMeta(ctx), l, &lines[i])
		}
		resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeDelivery, d.ID, "lines",
			apiresource.NewList(lines, apiresource.PageInfo{}))
	}
}

// deliveryFromProto builds the base Delivery from the detail-shaped proto. Expandable sub-objects are left nil and stashed by stashDeliveryMeta, so they appear only when asked for.
func deliveryFromProto(d *pb.DeliveryInfo) apiresource.Delivery {
	if d == nil {
		return apiresource.Delivery{}
	}

	return apiresource.Delivery{
		ID:         d.Id,
		Object:     constants.ObjectTypeDelivery,
		Number:     d.Number,
		Status:     constants.DeliveryStatus(d.Status),
		AcceptedAt: grpcutil.TimestampToTimePtr(d.AcceptedAt),
		RejectedAt: grpcutil.TimestampToTimePtr(d.RejectedAt),
		CreatedAt:  grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(d.UpdatedAt),
	}
}

func stashDeliveryMeta(ctx context.Context, d *apiresource.Delivery, info *pb.DeliveryInfo) {
	if info == nil {
		return
	}
	meta := resourcekit.GetLoadMeta(ctx)

	stashDeliveryRelated(ctx, d.ID, info.PurchaseOrderId, info.PurchaseOrderNumber, info.PurchaseOrderStatus, info.ReceivingOrderId, info.ReceivingOrderNumber, info.ReceivingOrderStatus)

	if len(info.Lines) > 0 {
		lines := make([]apiresource.DeliveryLine, len(info.Lines))
		for i, l := range info.Lines {
			lines[i] = deliveryLineFromProto(l)
			stashDeliveryLineMeta(resourcekit.GetLoadMeta(ctx), l, &lines[i])
		}
		meta.Set(constants.ObjectTypeDelivery, d.ID, "lines",
			apiresource.NewList(lines, apiresource.PageInfo{}))
	}
}

func deliveryLineFromProto(l *pb.DeliveryLineInfo) apiresource.DeliveryLine {
	if l == nil {
		return apiresource.DeliveryLine{}
	}

	return apiresource.DeliveryLine{
		ID:     l.Id,
		Object: constants.ObjectTypeDeliveryLine,
		Quantity: &apiresource.Quantity{
			ID:           l.QuantityId,
			Object:       constants.ObjectTypeQuantity,
			Value:        l.QuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(l.QuantityValue, l.QuantityUnitAbbreviation, ""),
			// Unit left nil: expandable, loaded with real data via `lines.quantity.unit`; never fabricated.
		},
		// Item, unit cost, location and lot are expandable: left nil here and stashed for the
		// include resolver, so a line carries only what it is — a quantity and when it landed.
		AcceptedAt: grpcutil.TimestampToTimePtr(l.AcceptedAt),
		RejectedAt: grpcutil.TimestampToTimePtr(l.RejectedAt),
		CreatedAt:  grpcutil.TimestampToTime(l.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(l.UpdatedAt),
	}
}

// stashDeliveryLineMeta carries each expandable sub-object of a line from what the query returned, so the include resolver can reveal the ones a caller asked for.
func stashDeliveryLineMeta(meta *resourcekit.LoadMeta, l *pb.DeliveryLineInfo, line *apiresource.DeliveryLine) {
	if l == nil || line.Quantity == nil {
		return
	}
	if l.ItemId != nil && *l.ItemId != "" {
		meta.Set(constants.ObjectTypeDeliveryLine, line.ID, "item_id", *l.ItemId)
	}
	if l.LocationId != nil && *l.LocationId != "" {
		meta.Set(constants.ObjectTypeDeliveryLine, line.ID, "location_id", *l.LocationId)
	}
	if l.LotId != nil && *l.LotId != "" {
		meta.Set(constants.ObjectTypeDeliveryLine, line.ID, "lot", &apiresource.Lot{
			ID:        *l.LotId,
			Object:    constants.ObjectTypeLot,
			LotNumber: l.GetLotNumber(),
		})
	}
	// The unit the line's quantity is counted in, for `lines.quantity.unit`.
	meta.Set(constants.ObjectTypeQuantity, line.Quantity.ID, "unit_id", l.QuantityUnitId)
	// The purchase order line the goods were ordered on, resolved through the PO line loader.
	meta.Set(constants.ObjectTypeDeliveryLine, line.ID, "order_line_id", l.OrderLineId)

	// The delivery query returns the rate in full, so `lines.unit_cost` is a complete rate rather
	// than an id and a bare number. Its two units stay behind their own includes, the same as on
	// any other rate.
	meta.Set(constants.ObjectTypeDeliveryLine, line.ID, "unit_cost", &apiresource.Rate{
		ID:     l.UnitCostId,
		Object: constants.ObjectTypeRate,
		Value:  apiresource.NormalizeRateValue(l.UnitCostValue),
		DisplayValue: apiresource.FormatRateDisplay(
			l.UnitCostValue,
			l.UnitCostNumeratorUnitAbbreviation,
			l.UnitCostDenominatorUnitAbbreviation,
		),
		CreatedAt: grpcutil.TimestampToTime(l.UnitCostCreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(l.UnitCostUpdatedAt),
	})
	meta.Set(constants.ObjectTypeRate, l.UnitCostId, "numerator_unit_id", l.UnitCostNumeratorUnitId)
	meta.Set(constants.ObjectTypeRate, l.UnitCostId, "denominator_unit_id", l.UnitCostDenominatorUnitId)
}

func deliveryListFromProto(ctx context.Context, resp *pb.ListDeliveriesResponse) *apiresource.List[apiresource.Delivery] {
	if resp == nil {
		return apiresource.NewList[apiresource.Delivery](nil, apiresource.PageInfo{})
	}

	deliveries := make([]apiresource.Delivery, len(resp.Deliveries))
	for i, d := range resp.Deliveries {
		deliveries[i] = deliverySummaryFromProto(d)
		stashDeliverySummaryMeta(ctx, &deliveries[i], d)
	}

	return apiresource.NewList(deliveries, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}

// stashDeliveryRelated carries the documents this delivery sits between as record references, built from the ids, numbers and statuses the delivery query already returns. Each reference is complete, so a client reading `related` never has to fetch the document to label it.
func stashDeliveryRelated(ctx context.Context, deliveryID, purchaseOrderID, purchaseOrderNumber, purchaseOrderStatus string, receivingOrderID, receivingOrderNumber, receivingOrderStatus *string) {
	related := &apiresource.DeliveryRelated{Object: constants.ObjectTypeDeliveryRelated}
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
	if receivingOrderID != nil && *receivingOrderID != "" {
		ro := apiresource.NewRecord(*receivingOrderID, constants.RecordTypeReceivingOrder)
		if receivingOrderNumber != nil && *receivingOrderNumber != "" {
			ro.Number = receivingOrderNumber
		}
		if receivingOrderStatus != nil && *receivingOrderStatus != "" {
			ro.Status = receivingOrderStatus
		}
		related.ReceivingOrder = ro
	}
	if related.PurchaseOrder == nil && related.ReceivingOrder == nil {
		return
	}
	resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeDelivery, deliveryID, "related", related)
}
