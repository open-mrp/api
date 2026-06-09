package pickep

import (
	"context"
	"fmt"
	"time"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"

	"github.com/augno/api/services/api-gateway/internal/domain"
)

var pickEpSvcTracer = tracing.GetTracer("api-gateway.endpoints.picks.service")

var pickDetailIncludes = []string{"sales_order", "departments", "lines", "lines.sales_order_line"}

type PickSvc interface {
	ListPicks(ctx context.Context, req *ListPicksRequest) (*apiresource.List[apiresource.Pick], *apierror.APIError)
	GetPick(ctx context.Context, req *RetrievePickRequest) (*apiresource.Pick, *apierror.APIError)
	UpdatePick(ctx context.Context, req *UpdatePickRequest) (*apiresource.Pick, *apierror.APIError)
	PickAllLines(ctx context.Context, req *PickAllLinesRequest) (*apiresource.Pick, *apierror.APIError)
	VoidPick(ctx context.Context, req *VoidPickRequest) (*apiresource.Pick, *apierror.APIError)
	PackPick(ctx context.Context, req *PackPickRequest) (*apiresource.PackPickResponse, *apierror.APIError)
	GetPickShipments(ctx context.Context, req *GetPickShipmentsRequest) (*apiresource.PickShipmentsResponse, *apierror.APIError)
	UpdatePickLine(ctx context.Context, req *UpdatePickLineRequest) (*apiresource.PickLine, *apierror.APIError)
	PickPickLine(ctx context.Context, req *PickPickLineRequest) (*apiresource.PickLine, *apierror.APIError)
	VoidPickLine(ctx context.Context, req *VoidPickLineRequest) (*apiresource.PickLine, *apierror.APIError)
}

type PickSvcConfig struct {
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
		// Ask the backend to expand departments when requested (sales_order /
		// customer are resolved gateway-side from stashed FK ids).
		Includes: resourcekit.FilterIncludes(ctx, "departments"),
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
		picks[i] = pickSummaryFromProto(p)
		stashPickSummaryMeta(ctx, &picks[i], p)
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
	if v, ok := req.FinishedAt.Value(); ok {
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

func (m *pickSvcImpl) PackPick(ctx context.Context, req *PackPickRequest) (*apiresource.PackPickResponse, *apierror.APIError) {
	pbReq := &pb.PackPickRequest{Id: req.PickID, ShipmentCaseCount: req.ShipmentCaseCount}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.pack", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.PackPickResponse, error) {
			return m.coreClient.PackPick(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	pick := pickDetailFromProto(resp.Pick)
	populatePickGroups(&pick, resp.Pick)
	return &apiresource.PackPickResponse{Pick: &pick, ShipmentNumber: resp.ShipmentNumber}, nil
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
		ShipmentNumbers: resp.ShipmentNumbers,
		Count:           resp.Count,
	}, nil
}

func (m *pickSvcImpl) UpdatePickLine(ctx context.Context, req *UpdatePickLineRequest) (*apiresource.PickLine, *apierror.APIError) {
	pbReq := &pb.UpdatePickLineRequest{PickId: req.PickID, Id: req.PickLineID}
	if v, ok := req.QuantityValue.Value(); ok {
		pbReq.QuantityValue = &v
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

func pickSummaryFromProto(info *pb.PickSummaryInfo) apiresource.Pick {
	now := grpcutil.TimestampToTime(info.CreatedAt)
	s := apiresource.Pick{
		ID:        info.Id,
		Object:    constants.ObjectTypePick,
		Number:    info.Number,
		Priority:  constants.PriorityCode(info.PriorityCode),
		CreatedAt: now,
		UpdatedAt: grpcutil.TimestampToTime(info.UpdatedAt),
	}
	s.FinishedAt = grpcutil.TimestampToTimePtr(info.FinishedAt)
	return s
}

func stashPickSummaryMeta(ctx context.Context, s *apiresource.Pick, info *pb.PickSummaryInfo) {
	meta := resourcekit.GetLoadMeta(ctx)
	// sales_order and customer are expandable references: stash the FK ids so
	// LoadSalesOrders / LoadCustomers fetch real data on ?include=. Never fabricate.
	if info.SalesOrderId != "" {
		meta.Set(constants.ObjectTypePick, s.ID, "sales_order_id", info.SalesOrderId)
	}
	if info.CustomerId != "" {
		meta.Set(constants.ObjectTypePick, s.ID, "customer_id", info.CustomerId)
	}
	// Departments are populated on the summary only when the list includes them.
	if len(info.Departments) > 0 {
		now := grpcutil.TimestampToTime(info.CreatedAt)
		meta.Set(constants.ObjectTypePick, s.ID, "departments",
			apiresource.NewList(buildPickDepartments(info.Departments, now), apiresource.PageInfo{}))
	}
}

func pickDetailFromProto(info *pb.PickInfo) apiresource.Pick {
	now := grpcutil.TimestampToTime(info.CreatedAt)
	d := apiresource.Pick{
		ID:        info.Id,
		Object:    constants.ObjectTypePick,
		Number:    info.Number,
		Priority:  constants.PriorityCode(info.PriorityCode),
		CreatedAt: now,
		UpdatedAt: grpcutil.TimestampToTime(info.UpdatedAt),
	}
	d.FinishedAt = grpcutil.TimestampToTimePtr(info.FinishedAt)
	return d
}

func stashPickDetailMeta(ctx context.Context, d *apiresource.Pick, info *pb.PickInfo) {
	meta := resourcekit.GetLoadMeta(ctx)
	now := grpcutil.TimestampToTime(info.CreatedAt)

	// sales_order and customer are expandable references: stash the FK ids so
	// LoadSalesOrders / LoadCustomers fetch real data on ?include=. Never fabricate.
	if info.SalesOrderId != "" {
		meta.Set(constants.ObjectTypePick, d.ID, "sales_order_id", info.SalesOrderId)
	}
	if info.CustomerId != "" {
		meta.Set(constants.ObjectTypePick, d.ID, "customer_id", info.CustomerId)
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

	depts := buildPickDepartments(info.Departments, now)
	meta.Set(constants.ObjectTypePick, d.ID, "departments",
		apiresource.NewList(depts, apiresource.PageInfo{}))
}

// populatePickGroups fills the always-present Lines and Departments groups for
// action responses (e.g. pack) that do not run the include resolver pipeline.
// The loader-backed references (sales_order, customer, lines.sales_order_line)
// remain null, populated only via ?include= on the standard pick endpoints.
func populatePickGroups(d *apiresource.Pick, info *pb.PickInfo) {
	now := grpcutil.TimestampToTime(info.CreatedAt)

	if len(info.Lines) > 0 {
		lines := make([]apiresource.PickLine, len(info.Lines))
		for i, l := range info.Lines {
			lines[i] = pickLineDetailFromProto(l)
		}
		d.Lines = apiresource.NewList(lines, apiresource.PageInfo{})
	}

	d.Departments = apiresource.NewList(buildPickDepartments(info.Departments, now), apiresource.PageInfo{})
}

func pickLineDetailFromProto(info *pb.PickLineInfo) apiresource.PickLine {
	now := grpcutil.TimestampToTime(info.CreatedAt)
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
		CreatedAt: now,
		UpdatedAt: now,
	}
	d.PackedAt = grpcutil.TimestampToTimePtr(info.PackedAt)
	return d
}

func stashPickLineDetailMeta(ctx context.Context, d *apiresource.PickLine, info *pb.PickLineInfo) {
	resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypePickLine, d.ID, "sales_order_line",
		buildSalesOrderLineForPick(info))
}

// --- helpers ---

func buildPickDepartments(deps []*pb.PickDepartmentInfo, ts time.Time) []apiresource.Department {
	depts := make([]apiresource.Department, len(deps))
	for i, dep := range deps {
		name := dep.Name
		if name == "" {
			name = "Department"
		}
		depts[i] = apiresource.Department{
			ID:        dep.Id,
			Object:    constants.ObjectTypeDepartment,
			Name:      name,
			CreatedAt: ts,
			UpdatedAt: ts,
		}
	}
	return depts
}

// buildSalesOrderLineForPick builds a pre-built, new-shape SalesOrderLine
// reference from the pick line's identifying proto fields. There is no
// standalone sales-order-line loader, so the parent proto carries the line's
// identifying fields. Only the required base fields are set; the expandable
// money/quantity fields (Product, QuantityOrdered, UnitPrice, UnitCost, Totals)
// are left nil and are never fabricated.
func buildSalesOrderLineForPick(info *pb.PickLineInfo) *apiresource.SalesOrderLine {
	now := grpcutil.TimestampToTime(info.CreatedAt)
	sku := info.OrderLineSku
	if sku == "" {
		sku = "—"
	}
	var productDesc *string
	if info.OrderLineDescription != nil && *info.OrderLineDescription != "" {
		productDesc = info.OrderLineDescription
	}

	return &apiresource.SalesOrderLine{
		ID:                 info.SalesOrderLineId,
		Object:             constants.ObjectTypeSalesOrderLine,
		LineItemNumber:     info.OrderLineItemNumber,
		ProductSKU:         sku,
		ProductDescription: productDesc,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}
