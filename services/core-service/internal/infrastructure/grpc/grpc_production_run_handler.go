package grpc

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type productionRunGRPCHandler struct {
	pb.UnimplementedCoreProductionRunServiceServer

	productionRunSvc domain.ProductionRunSvc
}

func batchSummariesToProto(in []domain.ProductionRunBatchSummary) []*pb.ProductionRunBatchSummaryInfo {
	out := make([]*pb.ProductionRunBatchSummaryInfo, len(in))
	for i, s := range in {
		out[i] = &pb.ProductionRunBatchSummaryInfo{
			ItemId:           s.ItemID,
			ItemSku:          s.ItemSKU,
			UnitId:           s.UnitID,
			UnitAbbreviation: s.UnitAbbreviation,
			QuantityValue:    s.Quantity.String(),
			BatchCount:       s.BatchCount,
		}
	}
	return out
}

func productionRunToProto(pr *domain.ProductionRun) *pb.ProductionRunInfo {
	info := &pb.ProductionRunInfo{
		Id:                pr.ID,
		Number:            pr.Number,
		ResponsibleUserId: pr.ResponsibleUserID,
		BatchCount:        pr.BatchCount,
		BatchSummaries:    batchSummariesToProto(pr.BatchSummaries),
		CreatedAt:         timestamppb.New(pr.CreatedAt),
		UpdatedAt:         timestamppb.New(pr.UpdatedAt),
	}
	if pr.ResponsibleUserName != nil {
		info.ResponsibleUserName = pr.ResponsibleUserName
	}
	if pr.ResponsibleUserStatusCode != nil {
		info.ResponsibleUserStatusCode = pr.ResponsibleUserStatusCode
	}
	if pr.ResponsibleUserCreatedAt != nil {
		info.ResponsibleUserCreatedAt = timestamppb.New(*pr.ResponsibleUserCreatedAt)
	}
	if pr.ResponsibleUserUpdatedAt != nil {
		info.ResponsibleUserUpdatedAt = timestamppb.New(*pr.ResponsibleUserUpdatedAt)
	}
	if pr.StartedAt != nil {
		info.StartedAt = timestamppb.New(*pr.StartedAt)
	}
	if pr.CompletedAt != nil {
		info.CompletedAt = timestamppb.New(*pr.CompletedAt)
	}
	return info
}

func productionRunSummaryToProto(pr *domain.ProductionRunSummary) *pb.ProductionRunSummaryInfo {
	info := &pb.ProductionRunSummaryInfo{
		Id:                pr.ID,
		Number:            pr.Number,
		ResponsibleUserId: pr.ResponsibleUserID,
		BatchCount:        pr.BatchCount,
		BatchSummaries:    batchSummariesToProto(pr.BatchSummaries),
		CreatedAt:         timestamppb.New(pr.CreatedAt),
		UpdatedAt:         timestamppb.New(pr.UpdatedAt),
	}
	if pr.ResponsibleUserName != nil {
		info.ResponsibleUserName = pr.ResponsibleUserName
	}
	if pr.ResponsibleUserStatusCode != nil {
		info.ResponsibleUserStatusCode = pr.ResponsibleUserStatusCode
	}
	if pr.ResponsibleUserCreatedAt != nil {
		info.ResponsibleUserCreatedAt = timestamppb.New(*pr.ResponsibleUserCreatedAt)
	}
	if pr.ResponsibleUserUpdatedAt != nil {
		info.ResponsibleUserUpdatedAt = timestamppb.New(*pr.ResponsibleUserUpdatedAt)
	}
	if pr.StartedAt != nil {
		info.StartedAt = timestamppb.New(*pr.StartedAt)
	}
	if pr.CompletedAt != nil {
		info.CompletedAt = timestamppb.New(*pr.CompletedAt)
	}
	return info
}

func (h *productionRunGRPCHandler) ExportProductionRuns(ctx context.Context, req *pb.ExportProductionRunsRequest) (*pb.ExportProductionRunsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	job, apiErr := h.productionRunSvc.ExportProductionRuns(ctx, domain.ExportProductionRunsParams{Query: req.Query})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ExportProductionRunsResponse{Job: jobToProto(job)}, nil
}

func (h *productionRunGRPCHandler) ListProductionRuns(ctx context.Context, req *pb.ListProductionRunsRequest) (*pb.ListProductionRunsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	// Omitted means open runs only; "all" drops the filter.
	status := req.Status
	if status == nil {
		defaultStatus := string(constants.ProductionRunStatusOpen)
		status = &defaultStatus
	} else if *status == string(constants.ProductionRunStatusAll) {
		status = nil
	}

	params := domain.ListProductionRunsParams{
		Cursor:     req.Cursor,
		Limit:      req.Limit,
		Query:      req.Query,
		Status:     status,
		ItemIDs:    req.ItemIds,
		MachineIDs: req.MachineIds,
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
	}

	result, apiErr := h.productionRunSvc.ListProductionRuns(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	runs := make([]*pb.ProductionRunSummaryInfo, len(result.ProductionRuns))
	for i, pr := range result.ProductionRuns {
		runs[i] = productionRunSummaryToProto(pr)
	}

	return &pb.ListProductionRunsResponse{
		ProductionRuns: runs,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *productionRunGRPCHandler) GetProductionRun(ctx context.Context, req *pb.GetProductionRunRequest) (*pb.GetProductionRunResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.GetProductionRunParams{
		ProductionRunID: req.Id,
	}

	result, apiErr := h.productionRunSvc.GetProductionRun(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetProductionRunResponse{
		ProductionRun: productionRunToProto(result),
	}, nil
}

func (h *productionRunGRPCHandler) CreateProductionRun(ctx context.Context, req *pb.CreateProductionRunRequest) (*pb.CreateProductionRunResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	batches, apiErr := addBatchInputsFromProto(req.Batches)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	params := domain.CreateProductionRunParams{
		ResponsibleUserID: req.ResponsibleUserId,
		Batches:           batches,
	}

	result, apiErr := h.productionRunSvc.CreateProductionRun(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateProductionRunResponse{
		ProductionRun: productionRunToProto(result),
	}, nil
}

func (h *productionRunGRPCHandler) UpdateProductionRun(ctx context.Context, req *pb.UpdateProductionRunRequest) (*pb.UpdateProductionRunResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateProductionRunParams{
		ProductionRunID:   req.Id,
		Number:            req.Number,
		ResponsibleUserID: req.ResponsibleUserId,
	}

	result, apiErr := h.productionRunSvc.UpdateProductionRun(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateProductionRunResponse{
		ProductionRun: productionRunToProto(result),
	}, nil
}

func (h *productionRunGRPCHandler) DeleteProductionRun(ctx context.Context, req *pb.DeleteProductionRunRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.DeleteProductionRunParams{
		ProductionRunID: req.Id,
	}

	apiErr := h.productionRunSvc.DeleteProductionRun(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *productionRunGRPCHandler) BulkCreateProductionRuns(ctx context.Context, req *pb.BulkCreateProductionRunsRequest) (*pb.BulkCreateProductionRunsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	runs := make([]domain.BulkCreateProductionRunParams, len(req.ProductionRuns))
	for i, r := range req.ProductionRuns {
		if r == nil {
			return nil, contracts.NewMissingGRPCRequestDataError()
		}
		batches := make([]domain.BulkCreateBatchParams, len(r.Batches))
		for j, b := range r.Batches {
			batches[j] = domain.BulkCreateBatchParams{
				Item:             itemIdentifierFromProto(b.Item),
				QuantityValue:    b.QuantityValue,
				QuantityUnit:     unitIdentifierFromProto(b.QuantityUnit),
				SecondsValue:     b.SecondsValue,
				SecondsUnit:      unitIdentifierPtrFromProto(b.SecondsUnit),
				WasteValue:       b.WasteValue,
				WasteUnit:        unitIdentifierPtrFromProto(b.WasteUnit),
				ProductionStepID: b.ProductionStepId,
				ScanningStation:  objectIdentifierPtrFromProto(b.ScanningStation),
			}
		}
		runs[i] = domain.BulkCreateProductionRunParams{
			ResponsibleUserID: r.ResponsibleUserId,
			Batches:           batches,
		}
	}

	job, apiErr := h.productionRunSvc.BulkCreateProductionRuns(ctx, domain.BulkCreateProductionRunsParams{ProductionRuns: runs})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.BulkCreateProductionRunsResponse{Job: jobToProto(job)}, nil
}

func (h *productionRunGRPCHandler) AddBatchesToProductionRun(ctx context.Context, req *pb.AddBatchesToProductionRunRequest) (*pb.AddBatchesToProductionRunResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	batches, apiErr := addBatchInputsFromProto(req.Batches)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	params := domain.AddBatchesToProductionRunParams{
		ProductionRunID: req.ProductionRunId,
		Batches:         batches,
	}

	result, apiErr := h.productionRunSvc.AddBatchesToProductionRun(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbBatches := make([]*pb.BaseBatchInfo, len(result))
	for i, b := range result {
		pbBatches[i] = baseBatchToProto(b)
	}

	return &pb.AddBatchesToProductionRunResponse{
		Batches: pbBatches,
	}, nil
}

func (h *productionRunGRPCHandler) ListBatchesByProductionRun(ctx context.Context, req *pb.ListBatchesByProductionRunRequest) (*pb.ListBatchesByProductionRunResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListBatchesByProductionRunParams{
		ProductionRunID: req.ProductionRunId,
		Cursor:          req.Cursor,
		Limit:           req.Limit,
		SearchQuery:     req.Query,
		Scope:           req.Scope,
	}

	result, apiErr := h.productionRunSvc.ListBatchesByProductionRun(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbBatches := make([]*pb.BatchInfo, len(result.Batches))
	for i, b := range result.Batches {
		pbBatches[i] = batchToProto(b)
	}

	return &pb.ListBatchesByProductionRunResponse{
		Batches: pbBatches,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

// addBatchInputsFromProto parses planned batches, rejecting any value that is not a decimal
// rather than letting it become zero.
func addBatchInputsFromProto(in []*pb.AddBatchInput) ([]domain.AddBatchInput, *apierror.APIError) {
	parse := func(value, param string) (decimal.Decimal, *apierror.APIError) {
		d, err := decimal.NewFromString(value)
		if err != nil {
			return decimal.Decimal{}, apierror.NewValidationErrorWithParam(fmt.Sprintf("%q is not a decimal.", value), param)
		}
		return d, nil
	}

	batches := make([]domain.AddBatchInput, len(in))
	for i, b := range in {
		param := func(field string) string { return fmt.Sprintf("batches[%d].%s", i, field) }

		measure, apiErr := parse(b.QuantityValue, param("quantity_value"))
		if apiErr != nil {
			return nil, apiErr
		}
		input := domain.AddBatchInput{
			ItemID:            b.ItemId,
			Quantity:          domain.CreateQuantityParams{Measure: measure, UnitID: b.QuantityUnitId},
			ProductionStepID:  b.ProductionStepId,
			ScanningStationID: b.ScanningStationId,
			MachineIDs:        b.MachineIds,
		}

		// Seconds and waste are recorded only when their unit is given.
		if b.SecondsValue != nil && b.SecondsUnitId != nil {
			secMeasure, apiErr := parse(*b.SecondsValue, param("seconds_value"))
			if apiErr != nil {
				return nil, apiErr
			}
			input.Seconds = &domain.CreateQuantityParams{Measure: secMeasure, UnitID: *b.SecondsUnitId}
		}
		if b.WasteValue != nil && b.WasteUnitId != nil {
			wasteMeasure, apiErr := parse(*b.WasteValue, param("waste_value"))
			if apiErr != nil {
				return nil, apiErr
			}
			input.Waste = &domain.CreateQuantityParams{Measure: wasteMeasure, UnitID: *b.WasteUnitId}
		}

		batches[i] = input
	}
	return batches, nil
}
