package grpc

import (
	"context"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
)

func (h *gRPCHandler) ListSalesTargets(ctx context.Context, req *pb.ListSalesTargetsRequest) (*pb.ListSalesTargetsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.salesTargetSvc.ListSalesTargets(ctx, domain.ListSalesTargetsParams{
		SalesRepID: req.SalesRepId,
		Query:      req.Query,
		Limit:      req.Limit,
		Offset:     req.Offset,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	targets := make([]*pb.SalesTargetProto, len(result.SalesTargets))
	for i, t := range result.SalesTargets {
		targets[i] = salesTargetToProto(&t)
	}

	return &pb.ListSalesTargetsResponse{
		SalesTargets: targets,
		Total:        result.Total,
	}, nil
}

func (h *gRPCHandler) CreateSalesTarget(ctx context.Context, req *pb.CreateSalesTargetRequest) (*pb.CreateSalesTargetResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewValidationErrorWithParam("Invalid date format for start_date.", "start_date"))
	}
	endDate, err := time.Parse(time.RFC3339, req.EndDate)
	if err != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewValidationErrorWithParam("Invalid date format for end_date.", "end_date"))
	}

	target, apiErr := h.salesTargetSvc.CreateSalesTarget(ctx, domain.CreateSalesTargetParams{
		SalesRepID:   req.SalesRepId,
		StartDate:    startDate,
		EndDate:      endDate,
		AmountValue:  req.AmountValue,
		AmountUnitID: req.AmountUnitId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateSalesTargetResponse{
		SalesTarget: salesTargetToProto(target),
	}, nil
}

func (h *gRPCHandler) UpsertSalesTarget(ctx context.Context, req *pb.UpsertSalesTargetRequest) (*pb.UpsertSalesTargetResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewValidationErrorWithParam("Invalid date format for start_date.", "start_date"))
	}
	endDate, err := time.Parse(time.RFC3339, req.EndDate)
	if err != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewValidationErrorWithParam("Invalid date format for end_date.", "end_date"))
	}

	target, apiErr := h.salesTargetSvc.UpsertSalesTarget(ctx, domain.UpsertSalesTargetParams{
		TargetID:     req.TargetId,
		SalesRepID:   req.SalesRepId,
		StartDate:    startDate,
		EndDate:      endDate,
		AmountValue:  req.AmountValue,
		AmountUnitID: req.AmountUnitId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpsertSalesTargetResponse{
		SalesTarget: salesTargetToProto(target),
	}, nil
}

func salesTargetToProto(t *domain.SalesTarget) *pb.SalesTargetProto {
	if t == nil {
		return nil
	}
	return &pb.SalesTargetProto{
		Id:           t.ID,
		StartDate:    t.StartDate.Format(time.RFC3339),
		EndDate:      t.EndDate.Format(time.RFC3339),
		SalesRepId:   t.SalesRepID,
		AccountId:    t.AccountID,
		AmountValue:  t.AmountValue,
		AmountUnitId: t.AmountUnitID,
		AmountId:     t.AmountID,
		CreatedAt:    t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    t.UpdatedAt.Format(time.RFC3339),
	}
}
