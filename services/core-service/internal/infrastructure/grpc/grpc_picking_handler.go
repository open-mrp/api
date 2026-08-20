package grpc

import (
	"context"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type pickingGRPCHandler struct {
	pb.UnimplementedCorePickingServiceServer

	pickSvc     domain.PickSvc
	pickLineSvc domain.PickLineSvc
}

func pickToProto(p *domain.Pick) *pb.PickInfo {
	if p == nil {
		return nil
	}

	info := &pb.PickInfo{
		Id:               p.ID,
		Number:           p.Number,
		SalesOrderId:     p.SalesOrderID,
		SalesOrderNumber: p.SalesOrderNumber,
		CustomerId:       p.CustomerID,
		CustomerName:     p.CustomerName,
		CustomerNumber:   p.CustomerNumber,
		PriorityId:       p.PriorityID,
		PriorityCode:     string(p.PriorityCode),
		PriorityName:     p.PriorityName,
		CreatedAt:        timestamppb.New(p.CreatedAt),
		UpdatedAt:        timestamppb.New(p.UpdatedAt),
		LineCount:        p.LineCount,
		PickedCompletion: p.PickedCompletion,
		PackedCompletion: p.PackedCompletion,

		ShippingAddressId:            p.ShippingAddressID,
		ShippingAddressName:          p.ShippingAddressName,
		ShippingAddressPhone:         p.ShippingAddressPhone,
		ShippingAddressEmail:         p.ShippingAddressEmail,
		ShippingAddressIsDropShip:    p.ShippingAddressIsDropShip,
		ShippingAddressGeolocationId: p.ShippingAddressGeolocation,
		ShippingAddressStreetLine_1:  p.ShippingAddressStreetLine1,
		ShippingAddressStreetLine_2:  p.ShippingAddressStreetLine2,
		ShippingAddressLocality:      p.ShippingAddressLocality,
		ShippingAddressState:         p.ShippingAddressState,
		ShippingAddressPostalCode:    p.ShippingAddressPostalCode,
		ShippingAddressCountry:       p.ShippingAddressCountry,
		ShipmentIds:                  p.ShipmentIDs,
	}

	if p.PromisedAt != nil {
		info.PromisedAt = timestamppb.New(*p.PromisedAt)
	}

	if p.ShipByDate != nil {
		info.ShipByDate = timestamppb.New(*p.ShipByDate)
	}
	info.LeadTimeDays = p.LeadTimeDays
	info.TransitDays = p.TransitDays
	if p.LeadTimeSource != nil {
		info.LeadTimeSource = p.LeadTimeSource.StringPtr()
	}
	if p.TransitSource != nil {
		info.TransitSource = p.TransitSource.StringPtr()
	}

	if p.LastShippedAt != nil {
		info.LastShippedAt = timestamppb.New(*p.LastShippedAt)
	}

	if p.FinishedAt != nil {
		info.FinishedAt = timestamppb.New(*p.FinishedAt)
	}

	if p.Lines != nil {
		lines := make([]*pb.PickLineInfo, len(p.Lines))
		for i, l := range p.Lines {
			lines[i] = pickLineToProto(l)
		}
		info.Lines = lines
	}

	if p.Departments != nil {
		depts := make([]*pb.PickDepartmentInfo, len(p.Departments))
		for i, d := range p.Departments {
			depts[i] = pickDepartmentToProto(d)
		}
		info.Departments = depts
	}

	return info
}

func pickLineToProto(l *domain.PickLine) *pb.PickLineInfo {
	if l == nil {
		return nil
	}

	info := &pb.PickLineInfo{
		Id:                                   l.ID,
		PickId:                               l.PickID,
		SalesOrderLineId:                     l.SalesOrderLineID,
		QuantityId:                           l.QuantityID,
		QuantityValue:                        l.QuantityValue,
		QuantityUnitId:                       l.QuantityUnitID,
		QuantityUnitName:                     l.QuantityUnitName,
		QuantityUnitAbbreviation:             l.QuantityUnitAbbreviation,
		CreatedAt:                            timestamppb.New(l.CreatedAt),
		UpdatedAt:                            timestamppb.New(l.UpdatedAt),
		OrderLineItemNumber:                  l.OrderLineItemNumber,
		OrderLineSku:                         l.OrderLineSKU,
		OrderLineProductId:                   l.OrderLineProductID,
		OrderLineItemId:                      l.OrderLineItemID,
		OrderedQuantityId:                    l.OrderedQuantityID,
		OrderedQuantityValue:                 l.OrderedQuantityValue,
		OrderedQuantityUnitId:                l.OrderedQuantityUnitID,
		OrderedQuantityUnitName:              l.OrderedQuantityUnitName,
		OrderedQuantityUnitAbbreviation:      l.OrderedQuantityUnitAbbrev,
		UnitPriceId:                          l.UnitPriceID,
		UnitPriceValue:                       l.UnitPriceValue,
		UnitPriceNumeratorUnitId:             l.UnitPriceNumeratorUnitID,
		UnitPriceNumeratorUnitAbbreviation:   l.UnitPriceNumeratorUnitAbbreviation,
		UnitPriceDenominatorUnitId:           l.UnitPriceDenominatorUnitID,
		UnitPriceDenominatorUnitAbbreviation: l.UnitPriceDenominatorUnitAbbreviation,
	}

	if l.PackedAt != nil {
		info.PackedAt = timestamppb.New(*l.PackedAt)
	}

	if l.OrderLineDescription != nil {
		info.OrderLineDescription = l.OrderLineDescription
	}

	return info
}

func pickDepartmentToProto(d *domain.PickDepartment) *pb.PickDepartmentInfo {
	if d == nil {
		return nil
	}

	return &pb.PickDepartmentInfo{
		Id:   d.ID,
		Name: d.Name,
	}
}

// ListPicks returns a paginated list of picks.
func (h *pickingGRPCHandler) ListPicks(ctx context.Context, req *pb.ListPicksRequest) (*pb.ListPicksResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListPicksParams{
		Limit: req.Limit,
	}

	if req.Cursor != nil {
		params.Cursor = req.Cursor
	}
	if req.Query != nil {
		params.Query = req.Query
	}
	if req.Status != nil {
		params.Status = req.Status
	}
	if len(req.CustomerIds) > 0 {
		params.CustomerIDs = req.CustomerIds
	}
	if len(req.ProductLineIds) > 0 {
		params.ProductLineIDs = req.ProductLineIds
	}
	if len(req.CustomerGroupIds) > 0 {
		params.CustomerGroupIDs = req.CustomerGroupIds
	}
	if len(req.DepartmentIds) > 0 {
		params.DepartmentIDs = req.DepartmentIds
	}
	if req.StartDate != nil {
		params.StartDate = req.StartDate
	}
	if req.EndDate != nil {
		params.EndDate = req.EndDate
	}
	params.Includes = req.Includes
	params.Sort = constants.PickSort(req.Sort)

	result, apiErr := h.pickSvc.ListPicks(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	picks := make([]*pb.PickInfo, len(result.Picks))
	for i, p := range result.Picks {
		picks[i] = pickToProto(p)
	}

	return &pb.ListPicksResponse{
		Picks: picks,
		PageInfo: &pb.PageInfo{
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
		},
	}, nil
}

// GetPick returns a single pick by ID.
func (h *pickingGRPCHandler) GetPick(ctx context.Context, req *pb.GetPickRequest) (*pb.GetPickResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	pick, apiErr := h.pickSvc.GetPick(ctx, req.Id, req.Includes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetPickResponse{
		Pick: pickToProto(pick),
	}, nil
}

// UpdatePick updates a pick's mutable fields.
func (h *pickingGRPCHandler) UpdatePick(ctx context.Context, req *pb.UpdatePickRequest) (*pb.UpdatePickResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdatePickParams{
		PickID:   req.Id,
		Includes: req.Includes,
	}

	if req.Number != nil {
		params.Number = req.Number
	}

	if req.FinishedAt != nil {
		if *req.FinishedAt == "" {
			// Empty string means set to null
			params.FinishedAt = new(*time.Time)
		} else {
			t, err := time.Parse(time.RFC3339, *req.FinishedAt)
			if err == nil {
				tt := &t
				params.FinishedAt = &tt
			}
		}
	}

	pick, apiErr := h.pickSvc.UpdatePick(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdatePickResponse{
		Pick: pickToProto(pick),
	}, nil
}

// PickAllLines marks all lines on a pick as picked.
func (h *pickingGRPCHandler) PickAllLines(ctx context.Context, req *pb.PickAllLinesRequest) (*pb.PickAllLinesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	pick, apiErr := h.pickSvc.PickAllLines(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.PickAllLinesResponse{
		Pick: pickToProto(pick),
	}, nil
}

// VoidPick voids a pick.
func (h *pickingGRPCHandler) VoidPick(ctx context.Context, req *pb.VoidPickRequest) (*pb.VoidPickResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	pick, apiErr := h.pickSvc.VoidPick(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.VoidPickResponse{
		Pick: pickToProto(pick),
	}, nil
}

// Accepts a pack and returns the job tracking it.
func (h *pickingGRPCHandler) PackPick(ctx context.Context, req *pb.PackPickRequest) (*pb.PackPickResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	job, apiErr := h.pickSvc.PackPick(ctx, req.Id, req.ShipmentCaseCount)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.PackPickResponse{Job: jobToProto(job)}, nil
}

// GetPickShipments returns the shipment numbers associated with a pick.
func (h *pickingGRPCHandler) GetPickShipments(ctx context.Context, req *pb.GetPickShipmentsRequest) (*pb.GetPickShipmentsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.GetPickShipmentsParams{
		PickID: req.Id,
		Query:  req.Query,
	}
	if req.Limit != nil {
		params.Limit = *req.Limit
	}
	if req.Offset != nil {
		params.Offset = *req.Offset
	}

	result, apiErr := h.pickSvc.GetPickShipments(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetPickShipmentsResponse{
		ShipmentNumbers: result.ShipmentNumbers,
		Count:           result.Count,
	}, nil
}

// UpdatePickLine updates a pick line's mutable fields.
func (h *pickingGRPCHandler) UpdatePickLine(ctx context.Context, req *pb.UpdatePickLineRequest) (*pb.UpdatePickLineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdatePickLineParams{
		PickID:        req.PickId,
		PickLineID:    req.Id,
		QuantityValue: req.QuantityValue,
	}

	line, apiErr := h.pickLineSvc.UpdatePickLine(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdatePickLineResponse{
		PickLine: pickLineToProto(line),
	}, nil
}

// PickPickLine marks a single pick line as picked.
func (h *pickingGRPCHandler) PickPickLine(ctx context.Context, req *pb.PickPickLineRequest) (*pb.PickPickLineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	line, apiErr := h.pickLineSvc.PickPickLine(ctx, req.PickId, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.PickPickLineResponse{
		PickLine: pickLineToProto(line),
	}, nil
}

// VoidPickLine voids a single pick line.
func (h *pickingGRPCHandler) VoidPickLine(ctx context.Context, req *pb.VoidPickLineRequest) (*pb.VoidPickLineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	line, apiErr := h.pickLineSvc.VoidPickLine(ctx, req.PickId, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.VoidPickLineResponse{
		PickLine: pickLineToProto(line),
	}, nil
}
