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
	ListPicks(ctx context.Context, req *ListPicksRequest) (*apiresource.List[apiresource.PickSummary], *apierror.APIError)
	GetPick(ctx context.Context, req *RetrievePickRequest) (*apiresource.PickDetail, *apierror.APIError)
	UpdatePick(ctx context.Context, req *UpdatePickRequest) (*apiresource.PickDetail, *apierror.APIError)
	PickAllLines(ctx context.Context, req *PickAllLinesRequest) (*apiresource.PickDetail, *apierror.APIError)
	VoidPick(ctx context.Context, req *VoidPickRequest) (*apiresource.PickDetail, *apierror.APIError)
	PackPick(ctx context.Context, req *PackPickRequest) (*apiresource.PackPickResponse, *apierror.APIError)
	GetPickShipments(ctx context.Context, req *GetPickShipmentsRequest) (*apiresource.PickShipmentsResponse, *apierror.APIError)
	UpdatePickLine(ctx context.Context, req *UpdatePickLineRequest) (*apiresource.PickLineDetail, *apierror.APIError)
	PickPickLine(ctx context.Context, req *PickPickLineRequest) (*apiresource.PickLineDetail, *apierror.APIError)
	VoidPickLine(ctx context.Context, req *VoidPickLineRequest) (*apiresource.PickLineDetail, *apierror.APIError)
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

func (m *pickSvcImpl) ListPicks(ctx context.Context, req *ListPicksRequest) (*apiresource.List[apiresource.PickSummary], *apierror.APIError) {
	pbReq := &pb.ListPicksRequest{
		Limit: req.Limit,
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

	picks := make([]apiresource.PickSummary, len(resp.Picks))
	for i, p := range resp.Picks {
		picks[i] = pickSummaryFromProto(p)
		stashPickSummaryMeta(ctx, &picks[i], p)
	}
	return apiresource.NewList(picks, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *pickSvcImpl) GetPick(ctx context.Context, req *RetrievePickRequest) (*apiresource.PickDetail, *apierror.APIError) {
	pbReq := &pb.GetPickRequest{Id: req.PickID, Includes: pickDetailIncludes}

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

func (m *pickSvcImpl) UpdatePick(ctx context.Context, req *UpdatePickRequest) (*apiresource.PickDetail, *apierror.APIError) {
	pbReq := &pb.UpdatePickRequest{Id: req.PickID, Includes: pickDetailIncludes}
	if req.Number != nil {
		pbReq.Number = req.Number
	}
	if req.FinishedAt != nil {
		pbReq.FinishedAt = req.FinishedAt
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

func (m *pickSvcImpl) PickAllLines(ctx context.Context, req *PickAllLinesRequest) (*apiresource.PickDetail, *apierror.APIError) {
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

func (m *pickSvcImpl) VoidPick(ctx context.Context, req *VoidPickRequest) (*apiresource.PickDetail, *apierror.APIError) {
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
	populatePickDetailExpandable(&pick, resp.Pick)
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

func (m *pickSvcImpl) UpdatePickLine(ctx context.Context, req *UpdatePickLineRequest) (*apiresource.PickLineDetail, *apierror.APIError) {
	pbReq := &pb.UpdatePickLineRequest{PickId: req.PickID, Id: req.PickLineID}
	if req.QuantityValue != nil {
		pbReq.QuantityValue = req.QuantityValue
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

func (m *pickSvcImpl) PickPickLine(ctx context.Context, req *PickPickLineRequest) (*apiresource.PickLineDetail, *apierror.APIError) {
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

func (m *pickSvcImpl) VoidPickLine(ctx context.Context, req *VoidPickLineRequest) (*apiresource.PickLineDetail, *apierror.APIError) {
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

func pickSummaryFromProto(info *pb.PickSummaryInfo) apiresource.PickSummary {
	now := grpcutil.TimestampToTime(info.CreatedAt)
	s := apiresource.PickSummary{
		ID:        info.Id,
		Object:    constants.ObjectTypePick,
		Number:    info.Number,
		Customer:  stubPickCustomer(info.CustomerId, info.CustomerName, info.CustomerNumber, now),
		Priority:  apiresource.ExpandablePriorityStub("", constants.PriorityCode(info.PriorityCode), info.PriorityName, now),
		CreatedAt: now,
		UpdatedAt: grpcutil.TimestampToTime(info.UpdatedAt),
	}
	s.FinishedAt = grpcutil.TimestampToTimePtr(info.FinishedAt)
	return s
}

func stashPickSummaryMeta(ctx context.Context, s *apiresource.PickSummary, info *pb.PickSummaryInfo) {
	meta := resourcekit.GetLoadMeta(ctx)
	now := grpcutil.TimestampToTime(info.CreatedAt)
	meta.Set(constants.ObjectTypePick, s.ID, "sales_order",
		buildPickSalesOrderStub(info.SalesOrderId, info.SalesOrderNumber, info.PriorityCode, info.PriorityName, now))
}

func pickDetailFromProto(info *pb.PickInfo) apiresource.PickDetail {
	now := grpcutil.TimestampToTime(info.CreatedAt)
	d := apiresource.PickDetail{
		ID:        info.Id,
		Object:    constants.ObjectTypePick,
		Number:    info.Number,
		Customer:  stubPickCustomer(info.CustomerId, info.CustomerName, info.CustomerNumber, now),
		Priority:  apiresource.ExpandablePriorityStub("", constants.PriorityCode(info.PriorityCode), info.PriorityName, now),
		CreatedAt: now,
		UpdatedAt: grpcutil.TimestampToTime(info.UpdatedAt),
	}
	d.FinishedAt = grpcutil.TimestampToTimePtr(info.FinishedAt)
	return d
}

func stashPickDetailMeta(ctx context.Context, d *apiresource.PickDetail, info *pb.PickInfo) {
	meta := resourcekit.GetLoadMeta(ctx)
	now := grpcutil.TimestampToTime(info.CreatedAt)

	meta.Set(constants.ObjectTypePick, d.ID, "sales_order",
		buildPickSalesOrderStub(info.SalesOrderId, info.SalesOrderNumber, info.PriorityCode, info.PriorityName, now))

	if len(info.Lines) > 0 {
		lines := make([]apiresource.PickLineDetail, len(info.Lines))
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

func populatePickDetailExpandable(d *apiresource.PickDetail, info *pb.PickInfo) {
	now := grpcutil.TimestampToTime(info.CreatedAt)
	d.SalesOrder = buildPickSalesOrderStub(info.SalesOrderId, info.SalesOrderNumber, info.PriorityCode, info.PriorityName, now)

	if len(info.Lines) > 0 {
		lines := make([]apiresource.PickLineDetail, len(info.Lines))
		for i, l := range info.Lines {
			lines[i] = pickLineDetailFromProto(l)
			lines[i].SalesOrderLine = buildSalesOrderLineForPick(l)
		}
		d.Lines = apiresource.NewList(lines, apiresource.PageInfo{})
	}

	d.Departments = apiresource.NewList(buildPickDepartments(info.Departments, now), apiresource.PageInfo{})
}

func pickLineDetailFromProto(info *pb.PickLineInfo) apiresource.PickLineDetail {
	now := grpcutil.TimestampToTime(info.CreatedAt)
	d := apiresource.PickLineDetail{
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
			Unit: stubUnitForPickLine(
				info.QuantityUnitId,
				info.QuantityUnitName,
				info.QuantityUnitAbbreviation,
				now,
			),
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
			Unit: stubUnitForPickLine(
				info.OrderedQuantityUnitId,
				info.OrderedQuantityUnitName,
				info.OrderedQuantityUnitAbbreviation,
				now,
			),
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	d.PackedAt = grpcutil.TimestampToTimePtr(info.PackedAt)
	return d
}

func stashPickLineDetailMeta(ctx context.Context, d *apiresource.PickLineDetail, info *pb.PickLineInfo) {
	resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypePickLine, d.ID, "sales_order_line",
		buildSalesOrderLineForPick(info))
}

// --- helpers ---

func stubPickCustomer(id, name, number string, now time.Time) *apiresource.Customer {
	return &apiresource.Customer{
		ID:               id,
		Object:           constants.ObjectTypeCustomer,
		Name:             name,
		Number:           number,
		Status:           constants.AccountStatusCodeNormal,
		CommissionPolicy: constants.CommissionPolicyApplied,
		EDIStatus:        constants.EDIStatusDisabled,
		RelationshipType: constants.CustomerRelationshipTypeStandalone,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func buildPickSalesOrderStub(salesOrderID, salesOrderNumber, priorityCode, priorityName string, now time.Time) *apiresource.SalesOrderDetail {
	return &apiresource.SalesOrderDetail{
		ID:     salesOrderID,
		Object: constants.ObjectTypeSalesOrder,
		Number: salesOrderNumber,
		Status: &apiresource.SalesOrderStatusDetail{
			Code:   string(constants.SalesOrderStatusCodeIssued),
			Object: constants.ObjectTypeSalesOrderStatus,
			Name:   "Issued",
		},
		Type: &apiresource.SalesOrderType{
			Code:   "standard",
			Object: constants.ObjectTypeSalesOrderType,
			Name:   "Standard",
		},
		Priority:  apiresource.ExpandablePriorityStub("", constants.PriorityCode(priorityCode), priorityName, now),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

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

func stubUnitForPickLine(id, name, abbr string, ts time.Time) *apiresource.Unit {
	if id == "" {
		id = "un_unknown"
	}
	if name == "" {
		name = abbr
	}
	if abbr == "" {
		abbr = "—"
	}
	return &apiresource.Unit{
		ID:                id,
		Object:            constants.ObjectTypeUnit,
		Name:              name,
		Abbreviation:      abbr,
		Type:              constants.UnitTypeQuantity,
		RatioNumerator:    "1",
		RatioDenominator:  "1",
		OffsetNumerator:   "0",
		OffsetDenominator: "1",
		CreatedAt:         ts,
		UpdatedAt:         ts,
	}
}

func buildSalesOrderLineForPick(info *pb.PickLineInfo) *apiresource.SalesOrderLineDetail {
	now := grpcutil.TimestampToTime(info.CreatedAt)
	sku := info.OrderLineSku
	if sku == "" {
		sku = "—"
	}
	var productDesc *string
	if info.OrderLineDescription != nil && *info.OrderLineDescription != "" {
		productDesc = info.OrderLineDescription
	}

	unitPriceValue := info.UnitPriceValue
	if unitPriceValue == "" {
		unitPriceValue = "0"
	}
	unitPriceID := info.UnitPriceId
	if unitPriceID == "" {
		unitPriceID = "ra_stub"
	}

	return &apiresource.SalesOrderLineDetail{
		ID:                 info.SalesOrderLineId,
		Object:             constants.ObjectTypeSalesOrderLine,
		LineItemNumber:     info.OrderLineItemNumber,
		ProductSKU:         sku,
		ProductDescription: productDesc,
		QuantityOrdered: &apiresource.Quantity{
			ID:     info.OrderedQuantityId,
			Object: constants.ObjectTypeQuantity,
			Value:  info.OrderedQuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(
				info.OrderedQuantityValue,
				info.OrderedQuantityUnitAbbreviation,
				string(constants.UnitTypeQuantity),
			),
			Unit: stubUnitForPickLine(
				info.OrderedQuantityUnitId,
				info.OrderedQuantityUnitName,
				info.OrderedQuantityUnitAbbreviation,
				now,
			),
		},
		UnitPrice: &apiresource.Rate{
			ID:     unitPriceID,
			Object: constants.ObjectTypeRate,
			Value:  unitPriceValue,
			NumeratorUnit: stubUnitForPickLine(
				info.UnitPriceNumeratorUnitId,
				info.UnitPriceNumeratorUnitAbbreviation,
				info.UnitPriceNumeratorUnitAbbreviation,
				now,
			),
			DenominatorUnit: stubUnitForPickLine(
				info.UnitPriceDenominatorUnitId,
				info.UnitPriceDenominatorUnitAbbreviation,
				info.UnitPriceDenominatorUnitAbbreviation,
				now,
			),
			DisplayValue: apiresource.FormatRateDisplayValue(
				unitPriceValue,
				info.UnitPriceNumeratorUnitAbbreviation,
				"",
				info.UnitPriceDenominatorUnitAbbreviation,
			),
			CreatedAt: now,
			UpdatedAt: now,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}
