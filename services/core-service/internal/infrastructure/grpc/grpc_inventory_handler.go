package grpc

import (
	"context"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
)

type inventoryGRPCHandler struct {
	pb.UnimplementedCoreInventoryServiceServer

	itemSvc domain.ItemSvc
}

// RegisterInventoryService serves CoreInventoryService from the item service, which owns the inventory ledger writes.
func RegisterInventoryService(server *grpc.Server, itemSvc domain.ItemSvc) {
	pb.RegisterCoreInventoryServiceServer(server, &inventoryGRPCHandler{itemSvc: itemSvc})
}

func parseMeasure(param, value string) (decimal.Decimal, *apierror.APIError) {
	d, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, apierror.NewFieldError(param, apierror.CodeInvalidFormat, param+" must be a decimal number.")
	}
	return d, nil
}

func reasonFromProto(r *string) *constants.InventoryMovementReason {
	if r == nil {
		return nil
	}
	reason := constants.InventoryMovementReason(*r)
	return &reason
}

func (h *inventoryGRPCHandler) CreateInventoryMovement(ctx context.Context, req *pb.CreateInventoryMovementRequest) (*pb.CreateInventoryMovementResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	measure, apiErr := parseMeasure("quantity.value", req.Measure)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	movement, apiErr := h.itemSvc.CreateInventoryMovement(ctx, domain.CreateInventoryMovementParams{
		ItemID:     req.ItemId,
		Reconcile:  req.Reconcile,
		Measure:    measure,
		UnitID:     req.UnitId,
		LocationID: req.LocationId,
		LotID:      req.LotId,
		LotNumber:  req.LotNumber,
		CustomerID: req.CustomerId,
		Reason:     reasonFromProto(req.Reason),
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.CreateInventoryMovementResponse{Movement: inventoryChangeLogToProto(movement)}, nil
}

func (h *inventoryGRPCHandler) BulkCreateInventoryMovements(ctx context.Context, req *pb.BulkCreateInventoryMovementsRequest) (*pb.BulkCreateInventoryMovementsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	rows := make([]domain.BulkCreateInventoryMovementRow, len(req.Rows))
	var rowErrs apierror.RowErrors
	for i, r := range req.Rows {
		measure, err := decimal.NewFromString(r.Measure)
		if err != nil {
			rowErrs.AddField(i, "quantity.value", apierror.CodeInvalidFormat, "must be a decimal number")
		}
		rows[i] = domain.BulkCreateInventoryMovementRow{
			ItemID: r.ItemId, SKU: r.Sku, Measure: measure, UnitID: r.UnitId,
			LocationID: r.LocationId, LotNumber: r.LotNumber, Reason: reasonFromProto(r.Reason),
		}
	}
	if apiErr := rowErrs.Summary("movements"); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	job, apiErr := h.itemSvc.BulkCreateInventoryMovements(ctx, domain.BulkCreateInventoryMovementsParams{Reconcile: req.Reconcile, Rows: rows})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.BulkCreateInventoryMovementsResponse{Job: jobToProto(job)}, nil
}

func (h *inventoryGRPCHandler) ListInventoryLevels(ctx context.Context, req *pb.ListInventoryLevelsRequest) (*pb.ListInventoryLevelsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	params := domain.ListInventoryLevelsParams{
		Cursor:      req.Cursor,
		Limit:       req.Limit,
		ItemIDs:     req.ItemIds,
		LocationIDs: req.LocationIds,
		SiteIDs:     req.SiteIds,
	}
	if req.AsOf != nil {
		t := req.AsOf.AsTime()
		params.AsOf = &t
	}
	result, apiErr := h.itemSvc.ListInventoryLevels(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	levels := make([]*pb.InventoryLevelInfo, len(result.Levels))
	for i, l := range result.Levels {
		levels[i] = stockLevelToProto(l)
	}
	return &pb.ListInventoryLevelsResponse{
		Levels: levels,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func decimalStringPtr(d *decimal.Decimal) *string {
	if d == nil {
		return nil
	}
	s := d.String()
	return &s
}

func stockLevelToProto(l *domain.StockLevel) *pb.InventoryLevelInfo {
	return &pb.InventoryLevelInfo{
		ItemId:             l.ItemID,
		LocationId:         l.LocationID,
		UnitId:             l.UnitID,
		OnHand:             l.OnHand.String(),
		Reserved:           decimalStringPtr(l.Reserved),
		AvailableToPromise: decimalStringPtr(l.AvailableToPromise),
		Short:              decimalStringPtr(l.Short),
	}
}
