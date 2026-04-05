package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *gRPCHandler) PullEDIOrders(ctx context.Context, req *pb.PullEDIOrdersRequest) (*pb.PullEDIOrdersResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.ediSvc.PullOrders(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.PullEDIOrdersResponse{
		Message: "EDI pull-orders operation completed successfully.",
	}, nil
}

func (h *gRPCHandler) ResubmitEDIInvoice(ctx context.Context, req *pb.ResubmitEDIInvoiceRequest) (*pb.ResubmitEDIInvoiceResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.ediSvc.ResubmitInvoice(ctx, req.InvoiceId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ResubmitEDIInvoiceResponse{
		Message: "Invoice resubmitted via EDI successfully.",
	}, nil
}

// ---------------------------------------------------------------------------
// DC Location proto conversions and handlers
// ---------------------------------------------------------------------------

func dcLocationToProto(d *domain.DCLocation) *pb.DCLocationProto {
	return &pb.DCLocationProto{
		Id:           d.ID,
		Location:     d.Location,
		CustomerId:   d.AccountID,
		CustomerName: d.CustomerName,
		CreatedAt:    timestamppb.New(d.CreatedAt),
		UpdatedAt:    timestamppb.New(d.UpdatedAt),
	}
}

func ediRunToProto(e *domain.EDIRun) *pb.EDIRunProto {
	return &pb.EDIRunProto{
		Id:           e.ID,
		CompletedAt:  timestamppb.New(e.CompletedAt),
		HasSucceeded: e.HasSucceeded,
		CreatedAt:    timestamppb.New(e.CreatedAt),
		UpdatedAt:    timestamppb.New(e.UpdatedAt),
	}
}

func (h *gRPCHandler) ListDCLocations(ctx context.Context, req *pb.ListDCLocationsRequest) (*pb.ListDCLocationsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListDCLocationsParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.ediSvc.ListDCLocations(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbLocs := make([]*pb.DCLocationProto, len(result.DCLocations))
	for i, d := range result.DCLocations {
		pbLocs[i] = dcLocationToProto(d)
	}

	return &pb.ListDCLocationsResponse{
		DcLocations: pbLocs,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetDCLocation(ctx context.Context, req *pb.GetDCLocationRequest) (*pb.GetDCLocationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	loc, apiErr := h.ediSvc.GetDCLocation(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetDCLocationResponse{
		DcLocation: dcLocationToProto(loc),
	}, nil
}

func (h *gRPCHandler) CreateDCLocation(ctx context.Context, req *pb.CreateDCLocationRequest) (*pb.CreateDCLocationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateDCLocationParams{
		AccountID: req.CustomerId,
		Location:  req.Location,
	}

	loc, apiErr := h.ediSvc.CreateDCLocation(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateDCLocationResponse{
		DcLocation: dcLocationToProto(loc),
	}, nil
}

func (h *gRPCHandler) UpdateDCLocation(ctx context.Context, req *pb.UpdateDCLocationRequest) (*pb.UpdateDCLocationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateDCLocationParams{
		DCLocationID: req.Id,
		AccountID:    req.CustomerId,
		Location:     req.Location,
	}

	loc, apiErr := h.ediSvc.UpdateDCLocation(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateDCLocationResponse{
		DcLocation: dcLocationToProto(loc),
	}, nil
}

func (h *gRPCHandler) DeleteDCLocation(ctx context.Context, req *pb.DeleteDCLocationRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.ediSvc.DeleteDCLocation(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

// ---------------------------------------------------------------------------
// EDI Run handlers
// ---------------------------------------------------------------------------

func (h *gRPCHandler) ListEDIRuns(ctx context.Context, req *pb.ListEDIRunsRequest) (*pb.ListEDIRunsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListEDIRunsParams{
		Cursor:       req.Cursor,
		Limit:        req.Limit,
		HasSucceeded: req.HasSucceeded,
		Query:        req.Query,
	}

	result, apiErr := h.ediSvc.ListEDIRuns(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbRuns := make([]*pb.EDIRunProto, len(result.EDIRuns))
	for i, e := range result.EDIRuns {
		pbRuns[i] = ediRunToProto(e)
	}

	return &pb.ListEDIRunsResponse{
		EdiRuns: pbRuns,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetEDIRun(ctx context.Context, req *pb.GetEDIRunRequest) (*pb.GetEDIRunResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	run, apiErr := h.ediSvc.GetEDIRun(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetEDIRunResponse{
		EdiRun: ediRunToProto(run),
	}, nil
}
