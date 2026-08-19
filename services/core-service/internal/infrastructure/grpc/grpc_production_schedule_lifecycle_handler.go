package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/field"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/safeconv"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func deviationTypeToProto(t *domain.ScheduleDeviationType) *pb.ScheduleDeviationTypeInfo {
	return &pb.ScheduleDeviationTypeInfo{
		Id:        t.ID,
		Code:      t.Code,
		Name:      t.Name,
		CreatedAt: timestamppb.New(t.CreatedAt),
		UpdatedAt: timestamppb.New(t.UpdatedAt),
	}
}

func deviationToProto(d *domain.ProductionScheduleDeviation) *pb.ProductionScheduleDeviationInfo {
	info := &pb.ProductionScheduleDeviationInfo{
		Id:                   d.ID,
		ProductionScheduleId: d.ProductionScheduleID,
		DeviationTypeCode:    d.DeviationTypeCode,
		IsFrozenWeek:         d.IsFrozenWeek,
		DeltaQuantity:        d.DeltaQuantity,
		DeltaRunHours:        d.DeltaRunHours,
		ActorId:              d.ActorID,
		CreatedAt:            timestamppb.New(d.CreatedAt),
	}
	info.ProductionScheduleLineId = d.ProductionScheduleLineID
	info.WeekIndex = d.WeekIndex
	info.MachineId = d.MachineID
	info.ItemId = d.ItemID
	info.ReasonCode = d.ReasonCode
	info.ReasonNote = d.ReasonNote
	// The snapshots cross the wire as JSON text so the gateway can hand them to the client untouched; re-modelling them in proto would freeze the line shape twice.
	if len(d.BeforeJSON) > 0 {
		before := string(d.BeforeJSON)
		info.BeforeJson = &before
	}
	if len(d.AfterJSON) > 0 {
		after := string(d.AfterJSON)
		info.AfterJson = &after
	}
	return info
}

func (h *productionScheduleGRPCHandler) ListScheduleDeviationTypes(ctx context.Context, req *pb.ListScheduleDeviationTypesRequest) (*pb.ListScheduleDeviationTypesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	types, apiErr := h.productionScheduleSvc.ListScheduleDeviationTypes(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	out := make([]*pb.ScheduleDeviationTypeInfo, len(types))
	for i, t := range types {
		out[i] = deviationTypeToProto(t)
	}

	return &pb.ListScheduleDeviationTypesResponse{Types: out}, nil
}

func (h *productionScheduleGRPCHandler) ListProductionScheduleDeviations(ctx context.Context, req *pb.ListProductionScheduleDeviationsRequest) (*pb.ListProductionScheduleDeviationsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.productionScheduleSvc.ListProductionScheduleDeviations(ctx, domain.ListProductionScheduleDeviationsParams{
		ScheduleID: req.ScheduleId,
		Cursor:     req.Cursor,
		Limit:      req.Limit,
		FrozenOnly: req.FrozenOnly,
		Query:      req.Query,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	deviations := make([]*pb.ProductionScheduleDeviationInfo, len(result.Deviations))
	for i, d := range result.Deviations {
		deviations[i] = deviationToProto(d)
	}

	return &pb.ListProductionScheduleDeviationsResponse{
		Deviations: deviations,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *productionScheduleGRPCHandler) CreateProductionScheduleLine(ctx context.Context, req *pb.CreateProductionScheduleLineRequest) (*pb.CreateProductionScheduleLineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	line, apiErr := h.productionScheduleSvc.CreateProductionScheduleLine(ctx, domain.CreateProductionScheduleLineParams{
		ScheduleID: req.ScheduleId,
		WeekIndex:  req.WeekIndex,
		MachineID:  req.MachineId,
		ItemID:     req.ItemId,
		Quantity:   req.Quantity,
		Lots:       req.Lots,
		RunHours:   req.RunHours,
		ReasonCode: req.ReasonCode,
		ReasonNote: req.ReasonNote,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateProductionScheduleLineResponse{Line: scheduleLineToProto(line)}, nil
}

func (h *productionScheduleGRPCHandler) UpdateProductionScheduleLine(ctx context.Context, req *pb.UpdateProductionScheduleLineRequest) (*pb.UpdateProductionScheduleLineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	line, apiErr := h.productionScheduleSvc.UpdateProductionScheduleLine(ctx, domain.UpdateProductionScheduleLineParams{
		ScheduleID:    req.ScheduleId,
		LineID:        req.LineId,
		WeekIndex:     req.WeekIndex,
		MachineID:     req.MachineId,
		Quantity:      req.Quantity,
		Lots:          req.Lots,
		RunHours:      req.RunHours,
		SequenceIndex: req.SequenceIndex,
		StatusCode:    req.StatusCode,
		ReasonCode:    field.StringClearableFromProto(req.ReasonCode),
		ReasonNote:    req.ReasonNote,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateProductionScheduleLineResponse{Line: scheduleLineToProto(line)}, nil
}

func (h *productionScheduleGRPCHandler) DeleteProductionScheduleLine(ctx context.Context, req *pb.DeleteProductionScheduleLineRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	if apiErr := h.productionScheduleSvc.DeleteProductionScheduleLine(ctx, domain.DeleteProductionScheduleLineParams{
		ScheduleID: req.ScheduleId,
		LineID:     req.LineId,
		ReasonCode: req.ReasonCode,
		ReasonNote: req.ReasonNote,
	}); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *productionScheduleGRPCHandler) PublishProductionSchedule(ctx context.Context, req *pb.PublishProductionScheduleRequest) (*pb.PublishProductionScheduleResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	schedule, apiErr := h.productionScheduleSvc.PublishProductionSchedule(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.PublishProductionScheduleResponse{Schedule: scheduleToProto(schedule)}, nil
}

func (h *productionScheduleGRPCHandler) ArchiveProductionSchedule(ctx context.Context, req *pb.ArchiveProductionScheduleRequest) (*pb.ArchiveProductionScheduleResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	schedule, apiErr := h.productionScheduleSvc.ArchiveProductionSchedule(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ArchiveProductionScheduleResponse{Schedule: scheduleToProto(schedule)}, nil
}

func (h *productionScheduleGRPCHandler) DeleteProductionSchedule(ctx context.Context, req *pb.DeleteProductionScheduleRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	if apiErr := h.productionScheduleSvc.DeleteProductionSchedule(ctx, req.Id); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *productionScheduleGRPCHandler) ListProductionScheduleDerivedLines(ctx context.Context, req *pb.ListProductionScheduleDerivedLinesRequest) (*pb.ListProductionScheduleDerivedLinesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	lines, apiErr := h.productionScheduleSvc.ListProductionScheduleDerivedLines(ctx, domain.ListDerivedLinesParams{
		ScheduleID:    req.ScheduleId,
		DepartmentIDs: req.DepartmentIds,
		WeekIndex:     req.WeekIndex,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	out := make([]*pb.ProductionScheduleDerivedLineInfo, len(lines))
	for i, line := range lines {
		info := &pb.ProductionScheduleDerivedLineInfo{
			Id:                   line.ID,
			ProductionScheduleId: line.ProductionScheduleID,
			SourceLineId:         line.SourceLineID,
			ProductionStepId:     line.ProductionStepID,
			ItemId:               line.ItemID,
			WeekIndex:            line.WeekIndex,
			WeekStartDate:        timestamppb.New(line.WeekStartDate),
			Quantity:             line.Quantity,
			ExplosionDepth:       line.ExplosionDepth,
			OffsetWeeks:          line.OffsetWeeks,
			StatusCode:           line.StatusCode,
			CreatedAt:            timestamppb.New(line.CreatedAt),
			UpdatedAt:            timestamppb.New(line.UpdatedAt),
		}
		info.DepartmentId = line.DepartmentID
		info.PlannedUnitId = line.PlannedUnitID
		out[i] = info
	}

	return &pb.ListProductionScheduleDerivedLinesResponse{Lines: out}, nil
}

// unitOrNil keeps an absent unit absent rather than sending an empty string that reads as a unit named "".
func unitOrNil(unit string) *string {
	if unit == "" {
		return nil
	}
	return &unit
}

func releasedLineToProto(l domain.ReleasedScheduleLine) *pb.ReleasedScheduleLineInfo {
	batches := make([]*pb.ReleaseScheduleLineBatchInfo, 0, len(l.Batches))
	for _, b := range l.Batches {
		info := &pb.ReleaseScheduleLineBatchInfo{
			ItemId:   b.ItemID,
			Sku:      b.SKU,
			Quantity: b.Quantity,
		}
		// Empty on a preview of a batch that would be created; a carried-forward ticket already exists and names itself even there.
		if b.BatchID != "" {
			batchID := b.BatchID
			info.BatchId = &batchID
		}
		if b.CarriedForwardFrom != "" {
			from := b.CarriedForwardFrom
			info.CarriedForwardFrom = &from
		}
		batches = append(batches, info)
	}

	return &pb.ReleasedScheduleLineInfo{
		ProductionScheduleLineId: l.ProductionScheduleLineID,
		ItemId:                   l.ItemID,
		Sku:                      l.SKU,
		MachineId:                l.MachineID,
		MachineName:              l.MachineName,
		PlannedQuantity:          l.PlannedQuantity,
		LotUnits:                 l.LotUnits,
		Unit:                     unitOrNil(l.Unit),
		BatchCount:               safeconv.IntToInt32(len(l.Batches)),
		Batches:                  batches,
		CarriedForwardQuantity:   l.CarriedForwardQuantity,
	}
}

func releasedLinesToProto(lines []domain.ReleasedScheduleLine) []*pb.ReleasedScheduleLineInfo {
	out := make([]*pb.ReleasedScheduleLineInfo, 0, len(lines))
	for _, line := range lines {
		out = append(out, releasedLineToProto(line))
	}
	return out
}

func (h *productionScheduleGRPCHandler) ReleaseProductionScheduleWeek(ctx context.Context, req *pb.ReleaseProductionScheduleWeekRequest) (*pb.ReleaseProductionScheduleWeekResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.productionScheduleSvc.ReleaseProductionScheduleWeek(ctx, domain.ReleaseScheduleWeekParams{
		ProductionScheduleID: req.Id,
		WeekIndex:            req.WeekIndex,
		ResponsibleUserID:    req.ResponsibleUserId,
		ScanningStationID:    req.ScanningStationId,
		SkipCarryForward:     req.SkipCarryForward,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ReleaseProductionScheduleWeekResponse{
		ProductionRun:            productionRunToProto(result.ProductionRun),
		WeekIndex:                result.WeekIndex,
		WeekStartDate:            timestamppb.New(result.WeekStartDate),
		ReleasedLineCount:        result.ReleasedLineCount,
		BatchCount:               result.BatchCount,
		CarriedForwardBatchCount: result.CarriedForwardBatchCount,
		TotalQuantity:            result.TotalQuantity,
		Lines:                    releasedLinesToProto(result.Lines),
	}, nil
}

func (h *productionScheduleGRPCHandler) PreviewReleaseProductionScheduleWeek(ctx context.Context, req *pb.PreviewReleaseProductionScheduleWeekRequest) (*pb.ReleaseProductionScheduleWeekPreviewResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	preview, apiErr := h.productionScheduleSvc.PreviewReleaseProductionScheduleWeek(ctx, req.Id, req.WeekIndex, req.SkipCarryForward)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ReleaseProductionScheduleWeekPreviewResponse{
		WeekIndex:                preview.WeekIndex,
		WeekStartDate:            timestamppb.New(preview.WeekStartDate),
		LineCount:                preview.LineCount,
		BatchCount:               preview.BatchCount,
		CarriedForwardBatchCount: preview.CarriedForwardBatchCount,
		TotalQuantity:            preview.TotalQuantity,
		Lines:                    releasedLinesToProto(preview.Lines),
		IsReleasable:             preview.IsReleasable,
		BlockedReason:            preview.BlockedReason,
		ExistingProductionRunId:  preview.ExistingProductionRunID,
	}, nil
}
