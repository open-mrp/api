package pickep

import (
	"context"
	"fmt"

	jobep "github.com/open-mrp/api/services/api-gateway/endpoints/jobs"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/ptrutil"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
)

var pickEpSvcTracer = tracing.GetTracer("api-gateway.endpoints.picks.service")

var pickDetailIncludes = []string{"related.sales_order", "related.shipments", "lines", "lines.sales_order_line"}

type PickSvc interface {
	ListPicks(ctx context.Context, req *ListPicksRequest) (*apiresource.List[apiresource.Pick], *apierror.APIError)
	GetPick(ctx context.Context, req *RetrievePickRequest) (*apiresource.Pick, *apierror.APIError)
	UpdatePick(ctx context.Context, req *UpdatePickRequest) (*apiresource.Pick, *apierror.APIError)
	PickAllLines(ctx context.Context, req *PickAllLinesRequest) (*apiresource.Pick, *apierror.APIError)
	VoidPick(ctx context.Context, req *VoidPickRequest) (*apiresource.Pick, *apierror.APIError)
	PackPick(ctx context.Context, req *PackPickRequest) (*apiresource.Job, *apierror.APIError)
	GetPickShipments(ctx context.Context, req *GetPickShipmentsRequest) (*apiresource.PickShipmentsResponse, *apierror.APIError)
	UpdatePickLine(ctx context.Context, req *UpdatePickLineRequest) (*apiresource.PickLine, *apierror.APIError)
	PickPickLine(ctx context.Context, req *PickPickLineRequest) (*apiresource.PickLine, *apierror.APIError)
	VoidPickLine(ctx context.Context, req *VoidPickLineRequest) (*apiresource.PickLine, *apierror.APIError)
}

type PickSvcConfig struct {
	// CoreClient (required) is the core-service picking gRPC client.
	CoreClient pb.CorePickingServiceClient
}

func (c *PickSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("pick endpoint service: core client is required")
	}
	return nil
}

type pickSvcImpl struct {
	coreClient pb.CorePickingServiceClient
}

func NewPickSvc(config *PickSvcConfig) PickSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &pickSvcImpl{coreClient: config.CoreClient}
}

func (m *pickSvcImpl) ListPicks(ctx context.Context, req *ListPicksRequest) (*apiresource.List[apiresource.Pick], *apierror.APIError) {
	pbReq := &pb.ListPicksRequest{
		Limit: req.Limit,
		Sort:  string(req.Sort),
		// List and detail return the same resource, so the backend gets the same include set
		// (related.sales_order / customer are resolved gateway-side from stashed FK ids).
		Includes: resourcekit.FilterIncludes(ctx, pickDetailIncludes...),
	}
	if req.Cursor != nil {
		pbReq.Cursor = req.Cursor
	}
	if req.Query != nil {
		pbReq.Query = req.Query
	}
	if req.Status != nil {
		pbReq.Status = req.Status
	}
	if len(req.CustomerIDs) > 0 {
		pbReq.CustomerIds = req.CustomerIDs
	}
	if len(req.ProductLineIDs) > 0 {
		pbReq.ProductLineIds = req.ProductLineIDs
	}
	if len(req.CustomerGroupIDs) > 0 {
		pbReq.CustomerGroupIds = req.CustomerGroupIDs
	}
	if len(req.DepartmentIDs) > 0 {
		pbReq.DepartmentIds = req.DepartmentIDs
	}
	if req.StartDate != nil {
		pbReq.StartDate = req.StartDate
	}
	if req.EndDate != nil {
		pbReq.EndDate = req.EndDate
	}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListPicksResponse, error) {
			return m.coreClient.ListPicks(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	picks := make([]apiresource.Pick, len(resp.Picks))
	for i, p := range resp.Picks {
		picks[i] = pickDetailFromProto(p)
		stashPickDetailMeta(ctx, &picks[i], p)
	}
	return apiresource.NewList(picks, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *pickSvcImpl) GetPick(ctx context.Context, req *RetrievePickRequest) (*apiresource.Pick, *apierror.APIError) {
	pbReq := &pb.GetPickRequest{Id: req.PickID, Includes: resourcekit.FilterIncludes(ctx, pickDetailIncludes...)}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetPickResponse, error) {
			return m.coreClient.GetPick(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := pickDetailFromProto(resp.Pick)
	stashPickDetailMeta(ctx, &result, resp.Pick)
	return &result, nil
}

func (m *pickSvcImpl) UpdatePick(ctx context.Context, req *UpdatePickRequest) (*apiresource.Pick, *apierror.APIError) {
	pbReq := &pb.UpdatePickRequest{Id: req.PickID, Includes: resourcekit.FilterIncludes(ctx, pickDetailIncludes...)}
	if v, ok := req.Number.Value(); ok {
		pbReq.Number = &v
	}
	// Core reads an empty string as "clear", so a null from the client maps onto that sentinel.
	if req.FinishedAt.IsClear() {
		cleared := ""
		pbReq.FinishedAt = &cleared
	} else if v, ok := req.FinishedAt.Value(); ok {
		pbReq.FinishedAt = &v
	}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdatePickResponse, error) {
			return m.coreClient.UpdatePick(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := pickDetailFromProto(resp.Pick)
	stashPickDetailMeta(ctx, &result, resp.Pick)
	return &result, nil
}

func (m *pickSvcImpl) PickAllLines(ctx context.Context, req *PickAllLinesRequest) (*apiresource.Pick, *apierror.APIError) {
	pbReq := &pb.PickAllLinesRequest{Id: req.PickID}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.pick_all_lines", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.PickAllLinesResponse, error) {
			return m.coreClient.PickAllLines(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := pickDetailFromProto(resp.Pick)
	stashPickDetailMeta(ctx, &result, resp.Pick)
	return &result, nil
}

func (m *pickSvcImpl) VoidPick(ctx context.Context, req *VoidPickRequest) (*apiresource.Pick, *apierror.APIError) {
	pbReq := &pb.VoidPickRequest{Id: req.PickID}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.void", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.VoidPickResponse, error) {
			return m.coreClient.VoidPick(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := pickDetailFromProto(resp.Pick)
	stashPickDetailMeta(ctx, &result, resp.Pick)
	return &result, nil
}

func (m *pickSvcImpl) PackPick(ctx context.Context, req *PackPickRequest) (*apiresource.Job, *apierror.APIError) {
	pbReq := &pb.PackPickRequest{Id: req.PickID, ShipmentCaseCount: req.ShipmentCaseCount}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.pack", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.PackPickResponse, error) {
			return m.coreClient.PackPick(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return jobep.JobFromProto(resp.GetJob()), nil
}

func (m *pickSvcImpl) GetPickShipments(ctx context.Context, req *GetPickShipmentsRequest) (*apiresource.PickShipmentsResponse, *apierror.APIError) {
	pbReq := &pb.GetPickShipmentsRequest{Id: req.PickID}
	if req.Query != nil {
		pbReq.Query = req.Query
	}
	if req.Limit != nil {
		pbReq.Limit = req.Limit
	}
	if req.Offset != nil {
		pbReq.Offset = req.Offset
	}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.get_shipments", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetPickShipmentsResponse, error) {
			return m.coreClient.GetPickShipments(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.PickShipmentsResponse{
		Object:          constants.ObjectTypePickShipmentsResponse,
		ShipmentNumbers: resp.ShipmentNumbers,
		Count:           resp.Count,
	}, nil
}

func (m *pickSvcImpl) UpdatePickLine(ctx context.Context, req *UpdatePickLineRequest) (*apiresource.PickLine, *apierror.APIError) {
	pbReq := &pb.UpdatePickLineRequest{
		PickId:        req.PickID,
		Id:            req.PickLineID,
		QuantityValue: req.QuantityValue.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.update_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdatePickLineResponse, error) {
			return m.coreClient.UpdatePickLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := pickLineDetailFromProto(resp.PickLine)
	stashPickLineDetailMeta(ctx, &result, resp.PickLine)
	return &result, nil
}

func (m *pickSvcImpl) PickPickLine(ctx context.Context, req *PickPickLineRequest) (*apiresource.PickLine, *apierror.APIError) {
	pbReq := &pb.PickPickLineRequest{PickId: req.PickID, Id: req.PickLineID}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.pick_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.PickPickLineResponse, error) {
			return m.coreClient.PickPickLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := pickLineDetailFromProto(resp.PickLine)
	stashPickLineDetailMeta(ctx, &result, resp.PickLine)
	return &result, nil
}

func (m *pickSvcImpl) VoidPickLine(ctx context.Context, req *VoidPickLineRequest) (*apiresource.PickLine, *apierror.APIError) {
	pbReq := &pb.VoidPickLineRequest{PickId: req.PickID, Id: req.PickLineID}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.void_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.VoidPickLineResponse, error) {
			return m.coreClient.VoidPickLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := pickLineDetailFromProto(resp.PickLine)
	stashPickLineDetailMeta(ctx, &result, resp.PickLine)
	return &result, nil
}

// --- inline presenter functions ---

func pickDetailFromProto(info *pb.PickInfo) apiresource.Pick {
	d := apiresource.Pick{
		ID:        info.Id,
		Object:    constants.ObjectTypePick,
		Number:    info.Number,
		Priority:  constants.PriorityCode(info.PriorityCode),
		CreatedAt: grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(info.UpdatedAt),

		LineCount: info.LineCount,
		Totals: &apiresource.PickTotals{
			Object: constants.ObjectTypePickTotals,
			Picked: apiresource.PickStageTotal{Object: constants.ObjectTypePickStageTotal, Completion: info.PickedCompletion},
			Packed: apiresource.PickStageTotal{Object: constants.ObjectTypePickStageTotal, Completion: info.PackedCompletion},
		},
		LastShippedAt: grpcutil.TimestampToTimePtr(info.LastShippedAt),
		PromisedAt:    grpcutil.TimestampToTimePtr(info.PromisedAt),
		ShipByDate:    grpcutil.TimestampToTimePtr(info.ShipByDate),
		LeadTimeDays:  info.LeadTimeDays,
		TransitDays:   info.TransitDays,
		ShipTo:        shipToFromPickProto(info),
	}
	d.FinishedAt = grpcutil.TimestampToTimePtr(info.FinishedAt)
	if info.LeadTimeSource != nil {
		v := constants.LeadTimeSource(*info.LeadTimeSource)
		d.LeadTimeSource = &v
	}
	if info.TransitSource != nil {
		v := constants.TransitSource(*info.TransitSource)
		d.TransitSource = &v
	}
	return d
}

// Builds the pick's ship-to from the order's address, denormalized onto the pick so the header
// renders without expanding the order. Nil when the order carries no address.
func shipToFromPickProto(info *pb.PickInfo) *apiresource.Address {
	if info.ShippingAddressId == "" {
		return nil
	}
	addr := &apiresource.Address{
		ID:     info.ShippingAddressId,
		Object: constants.ObjectTypeAddress,
		Name:   ptrutil.Deref(info.ShippingAddressName),
		Phone:  info.ShippingAddressPhone,
		Email:  info.ShippingAddressEmail,
		Type:   constants.AddressTypeStandard,
	}
	if info.GetShippingAddressIsDropShip() {
		addr.Type = constants.AddressTypeDropShip
	}
	if info.ShippingAddressGeolocationId != nil {
		addr.Geolocation = &apiresource.Geolocation{
			ID:          *info.ShippingAddressGeolocationId,
			Object:      constants.ObjectTypeGeolocation,
			StreetLine1: info.ShippingAddressStreetLine_1,
			StreetLine2: info.ShippingAddressStreetLine_2,
			Locality:    info.ShippingAddressLocality,
			State:       info.ShippingAddressState,
			PostalCode:  info.ShippingAddressPostalCode,
			Country:     ptrutil.Deref(info.ShippingAddressCountry),
		}
	}
	return addr
}

func stashPickDetailMeta(ctx context.Context, d *apiresource.Pick, info *pb.PickInfo) {
	meta := resourcekit.GetLoadMeta(ctx)

	// related.sales_order and customer are expandable references: stash the FK ids so
	// LoadSalesOrders / LoadCustomers fetch real data on ?include=. Never fabricate.
	if info.SalesOrderId != "" {
		meta.Set(constants.ObjectTypePick, d.ID, "sales_order_id", info.SalesOrderId)
	}
	if info.CustomerId != "" {
		meta.Set(constants.ObjectTypePick, d.ID, "customer_id", info.CustomerId)
	}
	if len(info.ShipmentIds) > 0 {
		meta.Set(constants.ObjectTypePick, d.ID, "related_shipment_ids", info.ShipmentIds)
	}

	if len(info.Lines) > 0 {
		lines := make([]apiresource.PickLine, len(info.Lines))
		for i, l := range info.Lines {
			lines[i] = pickLineDetailFromProto(l)
			stashPickLineDetailMeta(ctx, &lines[i], l)
		}
		meta.Set(constants.ObjectTypePick, d.ID, "lines",
			apiresource.NewList(lines, apiresource.PageInfo{}))
	}

}

func pickLineDetailFromProto(info *pb.PickLineInfo) apiresource.PickLine {
	d := apiresource.PickLine{
		ID:     info.Id,
		Object: constants.ObjectTypePickLine,
		Quantity: &apiresource.Quantity{
			ID:     info.QuantityId,
			Object: constants.ObjectTypeQuantity,
			Value:  info.QuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(
				info.QuantityValue,
				info.QuantityUnitAbbreviation,
				string(constants.UnitTypeQuantity),
			),
			// Unit left nil: expandable, loaded with real data via ?include=; never fabricated.
		},
		OrderedQuantity: &apiresource.Quantity{
			ID:     info.OrderedQuantityId,
			Object: constants.ObjectTypeQuantity,
			Value:  info.OrderedQuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(
				info.OrderedQuantityValue,
				info.OrderedQuantityUnitAbbreviation,
				string(constants.UnitTypeQuantity),
			),
			// Unit left nil: expandable, loaded with real data via ?include=; never fabricated.
		},
		CreatedAt: grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(info.UpdatedAt),
	}
	d.PackedAt = grpcutil.TimestampToTimePtr(info.PackedAt)
	return d
}

func stashPickLineDetailMeta(ctx context.Context, d *apiresource.PickLine, info *pb.PickLineInfo) {
	meta := resourcekit.GetLoadMeta(ctx)
	line := buildSalesOrderLineForPick(info)
	meta.Set(constants.ObjectTypePickLine, d.ID, "sales_order_line", line)

	// The item is the order line's, stashed so LoadItems resolves it on ?include=lines.item.
	if info.OrderLineItemId != nil && *info.OrderLineItemId != "" {
		meta.Set(constants.ObjectTypePickLine, d.ID, "item_id", *info.OrderLineItemId)
	}

	// The proto carries only the unit FK, so stash it on each quantity for LoadUnits to hydrate
	// on ?include=lines.quantity.unit. Without it the unit reads null and callers cannot tell
	// what the measure is denominated in.
	if info.QuantityUnitId != "" {
		meta.Set(constants.ObjectTypeQuantity, info.QuantityId, "unit_id", info.QuantityUnitId)
	}
	if info.OrderedQuantityUnitId != "" {
		meta.Set(constants.ObjectTypeQuantity, info.OrderedQuantityId, "unit_id", info.OrderedQuantityUnitId)
	}
	// Keyed by the order line, not the pick line, because the product loader runs against the
	// sales_order_line resource once the resolver recurses into it.
	if info.OrderLineProductId != nil && *info.OrderLineProductId != "" {
		meta.Set(constants.ObjectTypeSalesOrderLine, line.ID, "product_id", *info.OrderLineProductId)
	}
}

// --- helpers ---

// Builds the sales order line from what the pick line's proto carries, there being no standalone
// loader for it. The timestamps are the pick line's — the proto has none of the order line's own.
func buildSalesOrderLineForPick(info *pb.PickLineInfo) *apiresource.SalesOrderLine {
	createdAt := grpcutil.TimestampToTime(info.CreatedAt)

	return &apiresource.SalesOrderLine{
		ID:                 info.SalesOrderLineId,
		Object:             constants.ObjectTypeSalesOrderLine,
		LineItemNumber:     info.OrderLineItemNumber,
		ProductSKU:         info.OrderLineSku,
		ProductDescription: info.OrderLineDescription,
		CreatedAt:          createdAt,
		UpdatedAt:          createdAt,
	}
}
