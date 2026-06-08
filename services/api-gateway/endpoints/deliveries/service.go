package deliveryep

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
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DeliverySvc interface {
	ListDeliveries(ctx context.Context, req *ListDeliveriesRequest) (*apiresource.List[apiresource.Delivery], *apierror.APIError)
	GetDelivery(ctx context.Context, req *RetrieveDeliveryRequest) (*apiresource.Delivery, *apierror.APIError)
}

type DeliverySvcConfig struct {
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

// deliverySummaryFromProto builds the base Delivery from the list-shaped proto.
// The purchase_order reference is left nil; the FK id is stashed via
// stashDeliverySummaryMeta so LoadPurchaseOrders fetches real data on ?include=.
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
	// purchase_order is an expandable reference: stash the FK id so
	// LoadPurchaseOrders fetches real data on ?include=. Never fabricate.
	if info.PurchaseOrderId != "" {
		resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeDelivery, d.ID, "purchase_order_id", info.PurchaseOrderId)
	}
}

// deliveryFromProto builds the base Delivery from the detail-shaped proto.
// The purchase_order reference and lines are left nil; they are stashed via
// stashDeliveryMeta and populated only via ?include=.
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

	// purchase_order is an expandable reference: stash the FK id so
	// LoadPurchaseOrders fetches real data on ?include=. Never fabricate.
	if info.PurchaseOrderId != "" {
		meta.Set(constants.ObjectTypeDelivery, d.ID, "purchase_order_id", info.PurchaseOrderId)
	}

	if len(info.Lines) > 0 {
		lines := make([]apiresource.DeliveryLine, len(info.Lines))
		for i, l := range info.Lines {
			lines[i] = deliveryLineFromProto(l)
		}
		meta.Set(constants.ObjectTypeDelivery, d.ID, "lines",
			apiresource.NewList(lines, apiresource.PageInfo{}))
	}
}

func deliveryLineFromProto(l *pb.DeliveryLineInfo) apiresource.DeliveryLine {
	if l == nil {
		return apiresource.DeliveryLine{}
	}

	line := apiresource.DeliveryLine{
		ID:     l.Id,
		Object: constants.ObjectTypeDeliveryLine,
		Quantity: &apiresource.Quantity{
			ID:           l.QuantityId,
			Object:       constants.ObjectTypeQuantity,
			Value:        l.QuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(l.QuantityValue, l.QuantityUnitAbbreviation, ""),
			Unit: &apiresource.Unit{
				ID:     l.QuantityUnitId,
				Object: constants.ObjectTypeUnit,
			},
		},
		UnitCost: &apiresource.Rate{
			ID:     l.UnitCostId,
			Object: constants.ObjectTypeRate,
			Value:  l.UnitCostValue,
			NumeratorUnit: &apiresource.Unit{
				ID:     l.UnitCostNumeratorUnitId,
				Object: constants.ObjectTypeUnit,
			},
			DenominatorUnit: &apiresource.Unit{
				ID:     l.UnitCostDenominatorUnitId,
				Object: constants.ObjectTypeUnit,
			},
			DisplayValue: "",
		},
		CreatedAt: grpcutil.TimestampToTime(l.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(l.UpdatedAt),
	}

	if l.ItemId != nil {
		item := &apiresource.Item{
			ID:     *l.ItemId,
			Object: constants.ObjectTypeItem,
		}
		if l.ItemSku != nil {
			item.SKU = *l.ItemSku
		}
		line.Item = item
	}

	if l.LocationId != nil {
		loc := &apiresource.Location{
			ID:     *l.LocationId,
			Object: constants.ObjectTypeLocation,
		}
		if l.LocationName != nil {
			loc.Name = *l.LocationName
		}
		line.Location = loc
	}

	if l.LotId != nil {
		lot := &apiresource.Lot{
			ID:     *l.LotId,
			Object: constants.ObjectTypeLot,
		}
		if l.LotNumber != nil {
			lot.LotNumber = *l.LotNumber
		}
		line.Lot = lot
	}

	line.AcceptedAt = grpcutil.TimestampToTimePtr(l.AcceptedAt)
	line.RejectedAt = grpcutil.TimestampToTimePtr(l.RejectedAt)

	return line
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
