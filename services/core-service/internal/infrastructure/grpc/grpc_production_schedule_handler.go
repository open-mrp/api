package grpc

import (
	"context"
	"sort"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/scheduling"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/safeconv"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type productionScheduleGRPCHandler struct {
	pb.UnimplementedCoreProductionScheduleServiceServer

	productionScheduleSvc domain.ProductionScheduleSvc
	// Operating calendars ride on this service because the settings that select them are the same settings the solver is configured with.
	operatingCalendarSvc domain.OperatingCalendarSvc
}

func (h *productionScheduleGRPCHandler) PreviewProductionSchedule(ctx context.Context, req *pb.PreviewProductionScheduleRequest) (*pb.PreviewProductionScheduleResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.PreviewProductionScheduleParams{}
	if req.PlanningAsOf != nil {
		params.PlanningAsOf = req.PlanningAsOf.AsTime()
	}
	if req.HorizonWeeks != nil {
		params.HorizonWeeks = int(*req.HorizonWeeks)
	}
	if req.DemandBasis != nil {
		params.DemandBasis = *req.DemandBasis
	}

	out, apiErr := h.productionScheduleSvc.PreviewProductionSchedule(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	policies := make([]*pb.SchedulePolicyProto, len(out.Policies))
	for i, p := range out.Policies {
		policies[i] = &pb.SchedulePolicyProto{
			ItemId:                  p.ItemID,
			Sku:                     p.SKU,
			AnnualDemand:            p.AnnualDemand,
			WeeklyDemand:            p.WeeklyDemand,
			SecondsPerUnit:          p.SecondsPerUnit,
			UnitCost:                p.UnitCost,
			SetupCost:               p.SetupCost,
			HoldingCost:             p.HoldingCost,
			EoqUnits:                p.EOQUnits,
			ConstraintLeadTimeWeeks: p.ConstraintLeadTimeWeeks,
			FinishLeadTimeWeeks:     p.FinishLeadTimeWeeks,
			SafetyStockPrimary:      p.SafetyStockPrimary,
			SafetyStockDownstream:   p.SafetyStockDownstream,
			ReorderPoint:            p.ReorderPoint,
			OrderUpTo:               p.OrderUpTo,
			OnHandEchelon:           p.OnHandEchelon,
			OnHandGreige:            p.OnHandGreige,
			AverageGreigeInventory:  p.AverageGreigeInventory,
			MaxGreigeInventory:      p.MaxGreigeInventory,
			WeeksOfCover:            p.WeeksOfCover,
			AbcClass:                p.ABCClass,
			AnnualRunHours:          p.AnnualRunHours(),
			FulfillmentPolicyCode:   p.FulfillmentPolicy,
			PolicySourceCode:        p.PolicySource,
			FirmDemandUnits:         p.FirmDemandUnits,
			ForecastDemandUnits:     p.ForecastDemandUnits,
		}
	}

	campaigns := make([]*pb.ScheduleCampaignProto, len(out.Campaigns))
	for i, c := range out.Campaigns {
		campaigns[i] = &pb.ScheduleCampaignProto{
			ItemId:    c.ItemID,
			Sku:       c.SKU,
			MachineId: c.MachineID,
			WeekIndex: safeconv.IntToInt32(c.WeekIndex),
			Units:     c.Units,
			Lots:      safeconv.IntToInt32(c.Lots),
			RunHours:  c.RunHours,
		}
	}

	// Sorted so the response is byte-stable for the same plan; the map would otherwise reintroduce the nondeterminism the solver exists to remove.
	itemIDs := make([]string, 0, len(out.ProjectedOnHand))
	for itemID := range out.ProjectedOnHand {
		itemIDs = append(itemIDs, itemID)
	}
	sort.Strings(itemIDs)

	projections := make([]*pb.ScheduleProjectionProto, len(itemIDs))
	for i, itemID := range itemIDs {
		projections[i] = &pb.ScheduleProjectionProto{
			ItemId:       itemID,
			OnHandByWeek: out.ProjectedOnHand[itemID],
		}
	}

	appliedOverrides := make([]*pb.ScheduleAppliedOverrideProto, len(out.Diagnostics.AppliedOverrides))
	for i, o := range out.Diagnostics.AppliedOverrides {
		appliedOverrides[i] = &pb.ScheduleAppliedOverrideProto{
			OverrideId: o.OverrideID,
			ItemId:     o.ItemID,
			MonthStart: timestamppb.New(o.MonthStart),
			Before:     o.Before,
			After:      o.After,
			TypeCode:   o.TypeCode,
			ReasonCode: o.ReasonCode,
		}
	}

	return &pb.PreviewProductionScheduleResponse{
		SolverVersion: out.SolverVersion,
		PlanningAsOf:  timestamppb.New(out.PlanningAsOf),
		Policies:      policies,
		Campaigns:     campaigns,
		Projections:   projections,
		Diagnostics: &pb.ScheduleDiagnosticsProto{
			EoqCappedSkus:          out.Diagnostics.EOQCappedSKUs,
			UnschedulableSkus:      out.Diagnostics.UnschedulableSKUs,
			CapacityStarvedSkus:    out.Diagnostics.CapacityStarvedSKUs,
			ItemsWithoutRunRate:    out.Diagnostics.ItemsWithoutRunRate,
			ExcludedItemCount:      safeconv.IntToInt32(out.Diagnostics.ExcludedItemCount),
			ConstraintMachineCount: safeconv.IntToInt32(out.Diagnostics.ConstraintMachineCount),
			MeasuredBatchCount:     safeconv.IntToInt32(out.Diagnostics.MeasuredBatchCount),
			MachinesWithoutStep:    safeconv.IntToInt32(out.Diagnostics.MachinesWithoutStep),
			ChangeoverSlopeMinutes: out.Diagnostics.ChangeoverSlopeMinutes,
			AverageInputsAdded:     out.Diagnostics.AverageInputsAdded,
			AppliedOverrides:       appliedOverrides,
			FirmDemandUnits:        out.Diagnostics.FirmDemandUnits,
			UndatedFirmOrderCount:  safeconv.IntToInt32(out.Diagnostics.UndatedFirmOrderCount),
			MakeToOrderItemCount:   safeconv.IntToInt32(out.Diagnostics.MakeToOrderItemCount),
			AtRiskOrders:           scheduleAtRiskOrdersToProto(out.Diagnostics.AtRiskOrders),
			// The preview does not persist a plan, but it does solve both stages, so it reports both. A preview whose second stage read as empty would be worse than one that omitted it — it would look like a finding rather than an omission.
			Finishing:                    scheduleFinishingDiagnosticsToProto(out.Diagnostics.Finishing),
			FinishingMachineCount:        safeconv.IntToInt32(out.Diagnostics.FinishingMachineCount),
			FinishingCapacityIsEstimated: out.Diagnostics.FinishingCapacityIsEstimated,
		},
	}, nil
}

// scheduleFinishingDiagnosticsToProto maps stage two's account of itself, with every collection non-nil so an empty stage serializes as [] rather than null.
func scheduleFinishingDiagnosticsToProto(d scheduling.FinishingDiagnostics) *pb.ScheduleFinishingDiagnosticsProto {
	nonNilFloats := func(v []float64) []float64 {
		if v == nil {
			return []float64{}
		}
		return v
	}
	nonNilStrings := func(v []string) []string {
		if v == nil {
			return []string{}
		}
		return v
	}
	return &pb.ScheduleFinishingDiagnosticsProto{
		WeeklyCapacityHours: d.WeeklyCapacityHours,
		PlannedHoursByWeek:  nonNilFloats(d.PlannedHoursByWeek),
		UtilisationByWeek:   nonNilFloats(d.UtilisationByWeek),
		GreigeStarvedSkus:   nonNilStrings(d.GreigeStarvedSKUs),
		CapacityStarvedSkus: nonNilStrings(d.CapacityStarvedSKUs),
		ItemsWithoutRunRate: nonNilStrings(d.ItemsWithoutRunRate),
		UnusedGreigeUnits:   d.UnusedGreigeUnits,
		TotalPlannedUnits:   d.TotalPlannedUnits,
		LineCount:           safeconv.IntToInt32(d.LineCount),
	}
}

// scheduleAtRiskOrdersToProto maps the commitments the plan does not meet. Always a non-nil slice so an empty list serializes as [] rather than null.
func scheduleAtRiskOrdersToProto(orders []scheduling.AtRiskOrder) []*pb.ScheduleAtRiskOrderProto {
	out := make([]*pb.ScheduleAtRiskOrderProto, 0, len(orders))
	for _, o := range orders {
		out = append(out, &pb.ScheduleAtRiskOrderProto{
			SalesOrderId:     o.SalesOrderID,
			SalesOrderNumber: o.SalesOrderNumber,
			ItemId:           o.ItemID,
			Sku:              o.SKU,
			Units:            o.Units,
			DueWeek:          safeconv.IntToInt32(o.DueWeek),
			Reason:           o.Reason,
		})
	}
	return out
}

func scheduleToProto(s *domain.ProductionSchedule) *pb.ProductionScheduleInfo {
	if s == nil {
		return nil
	}
	info := &pb.ProductionScheduleInfo{
		Id:                    s.ID,
		Version:               s.Version,
		StatusCode:            s.StatusCode,
		Name:                  s.Name,
		PlanningAsOf:          timestamppb.New(s.PlanningAsOf),
		HorizonStartDate:      timestamppb.New(s.HorizonStartDate),
		HorizonEndDate:        timestamppb.New(s.HorizonEndDate),
		HorizonWeeks:          s.HorizonWeeks,
		FrozenWeeks:           s.FrozenWeeks,
		DemandBasisCode:       s.DemandBasisCode,
		GenerationSourceCode:  s.GenerationSourceCode,
		SolverVersion:         s.SolverVersion,
		SettingsSnapshot:      string(s.SettingsSnapshot),
		Diagnostics:           string(s.Diagnostics),
		ErrorMessage:          s.ErrorMessage,
		FrozenLineCount:       s.FrozenLineCount,
		FrozenPlannedQuantity: s.FrozenPlannedQuantity,
		GeneratedById:         s.GeneratedByID,
		PublishedById:         s.PublishedByID,
		SupersededById:        s.SupersededByID,
		CreatedAt:             timestamppb.New(s.CreatedAt),
		UpdatedAt:             timestamppb.New(s.UpdatedAt),
	}
	if s.FrozenThroughDate != nil {
		info.FrozenThroughDate = timestamppb.New(*s.FrozenThroughDate)
	}
	if s.PublishedAt != nil {
		info.PublishedAt = timestamppb.New(*s.PublishedAt)
	}
	return info
}

func scheduleLineToProto(l *domain.ProductionScheduleLine) *pb.ProductionScheduleLineInfo {
	return &pb.ProductionScheduleLineInfo{
		Id:                       l.ID,
		ProductionScheduleId:     l.ProductionScheduleID,
		WeekIndex:                l.WeekIndex,
		WeekStartDate:            timestamppb.New(l.WeekStartDate),
		MachineId:                l.MachineID,
		ProductionStepId:         l.ProductionStepID,
		DepartmentId:             l.DepartmentID,
		ItemId:                   l.ItemID,
		PlannedQuantity:          l.PlannedQuantity,
		PlannedUnitId:            l.PlannedUnitID,
		PlannedLots:              l.PlannedLots,
		PlannedLotUnits:          l.PlannedLotUnits,
		PlannedUnitAbbreviation:  l.PlannedUnitAbbreviation,
		ReleasedBatchCount:       l.ReleasedBatchCount,
		ScannedBatchCount:        l.ScannedBatchCount,
		ScannedQuantity:          l.ScannedQuantity,
		PlannedRunHours:          l.PlannedRunHours,
		PlannedChangeoverMinutes: l.PlannedChangeoverMinutes,
		SequenceIndex:            l.SequenceIndex,
		ProjectedOnHandBefore:    l.ProjectedOnHandBefore,
		ProjectedOnHandAfter:     l.ProjectedOnHandAfter,
		StatusCode:               l.StatusCode,
		SourceCode:               l.SourceCode,
		ReasonCode:               l.ReasonCode,
		IsFrozen:                 l.IsFrozen,
		ProductionRunId:          l.ProductionRunID,
		CreatedAt:                timestamppb.New(l.CreatedAt),
		UpdatedAt:                timestamppb.New(l.UpdatedAt),
	}
}

func scheduleItemPolicyToProto(p *domain.ProductionScheduleItemPolicy) *pb.ProductionScheduleItemPolicyInfo {
	return &pb.ProductionScheduleItemPolicyInfo{
		Id:                      p.ID,
		ProductionScheduleId:    p.ProductionScheduleID,
		ItemId:                  p.ItemID,
		Sku:                     p.SKU,
		UnitId:                  p.UnitID,
		UnitAbbreviation:        p.UnitAbbreviation,
		ProductionStepId:        p.ProductionStepID,
		PrimaryMachineId:        p.PrimaryMachineID,
		AnnualDemand:            p.AnnualDemand,
		WeeklyDemand:            p.WeeklyDemand,
		SecondsPerUnit:          p.SecondsPerUnit,
		UnitCost:                p.UnitCost,
		SetupCost:               p.SetupCost,
		HoldingCost:             p.HoldingCost,
		EoqUnits:                p.EOQUnits,
		ConstraintLeadTimeWeeks: p.ConstraintLeadTimeWeeks,
		FinishLeadTimeWeeks:     p.FinishLeadTimeWeeks,
		SigmaWeeklyPooled:       p.SigmaWeeklyPooled,
		SigmaDownstreamSum:      p.SigmaDownstreamSum,
		SafetyStockPrimary:      p.SafetyStockPrimary,
		SafetyStockDownstream:   p.SafetyStockDownstream,
		ReorderPoint:            p.ReorderPoint,
		OrderUpTo:               p.OrderUpTo,
		OnHandEchelon:           p.OnHandEchelon,
		OnHandGreige:            p.OnHandGreige,
		AverageGreigeInventory:  p.AverageGreigeInventory,
		MaxGreigeInventory:      p.MaxGreigeInventory,
		ProjectedOnHand:         p.ProjectedOnHand,
		WeeksOfCover:            p.WeeksOfCover,
		AnnualRunHours:          p.AnnualRunHours,
		AbcClass:                p.ABCClass,
		WasEoqCapped:            p.WasEOQCapped,
		WasCapacityStarved:      p.WasCapacityStarved,
		CreatedAt:               timestamppb.New(p.CreatedAt),
		UpdatedAt:               timestamppb.New(p.UpdatedAt),
		FulfillmentPolicyCode:   p.FulfillmentPolicyCode,
		PolicySourceCode:        p.PolicySourceCode,
		FirmDemandUnits:         p.FirmDemandUnits,
		ForecastDemandUnits:     p.ForecastDemandUnits,
	}
}

func (h *productionScheduleGRPCHandler) GenerateProductionSchedule(ctx context.Context, req *pb.GenerateProductionScheduleRequest) (*pb.GenerateProductionScheduleResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.GenerateProductionScheduleParams{Name: req.Name}
	if req.PlanningAsOf != nil {
		params.PlanningAsOf = req.PlanningAsOf.AsTime()
	}
	if req.HorizonWeeks != nil {
		params.HorizonWeeks = int(*req.HorizonWeeks)
	}
	if req.DemandBasis != nil {
		params.DemandBasis = *req.DemandBasis
	}

	schedule, apiErr := h.productionScheduleSvc.GenerateProductionSchedule(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.GenerateProductionScheduleResponse{Schedule: scheduleToProto(schedule)}, nil
}

// regenerateParamsFromProto is shared by the preview and the apply so the two can never answer different questions about the same request.
func regenerateParamsFromProto(req *pb.RegenerateProductionScheduleRequest) domain.RegenerateProductionScheduleParams {
	params := domain.RegenerateProductionScheduleParams{ScheduleID: req.Id}
	if req.MergeMode != nil {
		params.MergeMode = *req.MergeMode
	}
	if req.PlanningAsOf != nil {
		asOf := req.PlanningAsOf.AsTime()
		params.PlanningAsOf = &asOf
	}
	if req.HorizonWeeks != nil {
		params.HorizonWeeks = int(*req.HorizonWeeks)
	}
	if req.DemandBasis != nil {
		params.DemandBasis = *req.DemandBasis
	}
	return params
}

func (h *productionScheduleGRPCHandler) PreviewRegenerateProductionSchedule(ctx context.Context, req *pb.RegenerateProductionScheduleRequest) (*pb.PreviewRegenerateProductionScheduleResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	preview, apiErr := h.productionScheduleSvc.PreviewRegenerateProductionSchedule(ctx, regenerateParamsFromProto(req))
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	lines := make([]*pb.ScheduleDiffLineProto, len(preview.Lines))
	for i, line := range preview.Lines {
		lines[i] = &pb.ScheduleDiffLineProto{
			ChangeCode:       line.ChangeCode,
			ItemId:           line.ItemID,
			Sku:              line.SKU,
			MachineId:        line.MachineID,
			WeekIndex:        line.WeekIndex,
			CurrentQuantity:  line.CurrentQuantity,
			ProposedQuantity: line.ProposedQuantity,
			CurrentIsManual:  line.CurrentIsManual,
		}
	}

	return &pb.PreviewRegenerateProductionScheduleResponse{
		ScheduleId:           preview.ScheduleID,
		SolverVersion:        preview.SolverVersion,
		PlanningAsOf:         timestamppb.New(preview.PlanningAsOf),
		Lines:                lines,
		AddedCount:           preview.AddedCount,
		RemovedCount:         preview.RemovedCount,
		ChangedCount:         preview.ChangedCount,
		ManualLineCount:      preview.ManualLineCount,
		DiscardedManualCount: preview.DiscardedManualCount,
	}, nil
}

func (h *productionScheduleGRPCHandler) RegenerateProductionSchedule(ctx context.Context, req *pb.RegenerateProductionScheduleRequest) (*pb.RegenerateProductionScheduleResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	schedule, apiErr := h.productionScheduleSvc.RegenerateProductionSchedule(ctx, regenerateParamsFromProto(req))
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.RegenerateProductionScheduleResponse{Schedule: scheduleToProto(schedule)}, nil
}

func (h *productionScheduleGRPCHandler) GetProductionSchedule(ctx context.Context, req *pb.GetProductionScheduleRequest) (*pb.GetProductionScheduleResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	schedule, apiErr := h.productionScheduleSvc.GetProductionSchedule(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.GetProductionScheduleResponse{Schedule: scheduleToProto(schedule)}, nil
}

func (h *productionScheduleGRPCHandler) GetCurrentProductionSchedule(ctx context.Context, req *pb.GetCurrentProductionScheduleRequest) (*pb.GetProductionScheduleResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	schedule, apiErr := h.productionScheduleSvc.GetCurrentProductionSchedule(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	// A nil schedule is not an error: an account with no live plan is a normal state, and the gateway turns it into a 404 rather than a fabricated empty schedule.
	return &pb.GetProductionScheduleResponse{Schedule: scheduleToProto(schedule)}, nil
}

func (h *productionScheduleGRPCHandler) ListProductionSchedules(ctx context.Context, req *pb.ListProductionSchedulesRequest) (*pb.ListProductionSchedulesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.productionScheduleSvc.ListProductionSchedules(ctx, domain.ListProductionSchedulesParams{
		Cursor:      req.Cursor,
		Limit:       req.Limit,
		StatusCodes: req.StatusCodes,
		Query:       req.Query,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	schedules := make([]*pb.ProductionScheduleInfo, len(result.Schedules))
	for i, s := range result.Schedules {
		schedules[i] = scheduleToProto(s)
	}

	return &pb.ListProductionSchedulesResponse{
		Schedules: schedules,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *productionScheduleGRPCHandler) ListProductionScheduleLines(ctx context.Context, req *pb.ListProductionScheduleLinesRequest) (*pb.ListProductionScheduleLinesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListProductionScheduleLinesParams{
		ScheduleID: req.ProductionScheduleId,
		MachineIDs: req.MachineIds,
	}
	if req.WeekIndex != nil {
		params.WeekIndex = req.WeekIndex
	}

	lines, apiErr := h.productionScheduleSvc.ListProductionScheduleLines(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	out := make([]*pb.ProductionScheduleLineInfo, len(lines))
	for i, l := range lines {
		out[i] = scheduleLineToProto(l)
	}
	return &pb.ListProductionScheduleLinesResponse{Lines: out}, nil
}

func (h *productionScheduleGRPCHandler) ListProductionScheduleItemPolicies(ctx context.Context, req *pb.ListProductionScheduleItemPoliciesRequest) (*pb.ListProductionScheduleItemPoliciesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	policies, apiErr := h.productionScheduleSvc.ListProductionScheduleItemPolicies(ctx, req.ProductionScheduleId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	out := make([]*pb.ProductionScheduleItemPolicyInfo, len(policies))
	for i, p := range policies {
		out[i] = scheduleItemPolicyToProto(p)
	}
	return &pb.ListProductionScheduleItemPoliciesResponse{Policies: out}, nil
}

func (h *productionScheduleGRPCHandler) ListProductionScheduleFinishingLines(ctx context.Context, req *pb.ListProductionScheduleFinishingLinesRequest) (*pb.ListProductionScheduleFinishingLinesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	lines, apiErr := h.productionScheduleSvc.ListProductionScheduleFinishingLines(ctx, domain.ListProductionScheduleFinishingLinesParams{
		ScheduleID: req.ProductionScheduleId,
		WeekIndex:  req.WeekIndex,
		ItemID:     req.ItemId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	out := make([]*pb.ProductionScheduleFinishingLineInfo, len(lines))
	for i, l := range lines {
		out[i] = &pb.ProductionScheduleFinishingLineInfo{
			Id:                      l.ID,
			ProductionScheduleId:    l.ProductionScheduleID,
			WeekIndex:               l.WeekIndex,
			WeekStartDate:           timestamppb.New(l.WeekStartDate),
			ItemId:                  l.ItemID,
			Sku:                     l.SKU,
			GreigeItemId:            l.GreigeItemID,
			GreigeSku:               l.GreigeSKU,
			DepartmentId:            l.DepartmentID,
			ProductionStepId:        l.ProductionStepID,
			PlannedQuantity:         l.PlannedQuantity,
			PlannedUnitId:           l.PlannedUnitID,
			PlannedUnitAbbreviation: l.PlannedUnitAbbreviation,
			PlannedLots:             l.PlannedLots,
			PlannedLotUnits:         l.PlannedLotUnits,
			PlannedRunHours:         l.PlannedRunHours,
			GreigeConsumed:          l.GreigeConsumed,
			FirmUnits:               l.FirmUnits,
			ProjectedOnHandBefore:   l.ProjectedOnHandBefore,
			ProjectedOnHandAfter:    l.ProjectedOnHandAfter,
			StatusCode:              l.StatusCode,
			SourceCode:              l.SourceCode,
			IsFrozen:                l.IsFrozen,
			CreatedAt:               timestamppb.New(l.CreatedAt),
			UpdatedAt:               timestamppb.New(l.UpdatedAt),
		}
	}
	return &pb.ListProductionScheduleFinishingLinesResponse{Lines: out}, nil
}

func (h *productionScheduleGRPCHandler) ListProductionScheduleFinishedPolicies(ctx context.Context, req *pb.ListProductionScheduleFinishedPoliciesRequest) (*pb.ListProductionScheduleFinishedPoliciesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	policies, apiErr := h.productionScheduleSvc.ListProductionScheduleFinishedPolicies(ctx, req.ProductionScheduleId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	out := make([]*pb.ProductionScheduleFinishedPolicyInfo, len(policies))
	for i, p := range policies {
		out[i] = &pb.ProductionScheduleFinishedPolicyInfo{
			Id:                   p.ID,
			ProductionScheduleId: p.ProductionScheduleID,
			ItemId:               p.ItemID,
			Sku:                  p.SKU,
			GreigeItemId:         p.GreigeItemID,
			GreigeSku:            p.GreigeSKU,
			ProductLineId:        p.ProductLineID,
			AnnualDemand:         p.AnnualDemand,
			WeeklyDemand:         p.WeeklyDemand,
			SigmaWeekly:          p.SigmaWeekly,
			SafetyStock:          p.SafetyStock,
			ReorderPoint:         p.ReorderPoint,
			OnHand:               p.OnHand,
			WeeksOfCover:         p.WeeksOfCover,
			CreatedAt:            timestamppb.New(p.CreatedAt),
			UpdatedAt:            timestamppb.New(p.UpdatedAt),
		}
	}
	return &pb.ListProductionScheduleFinishedPoliciesResponse{Policies: out}, nil
}

func (h *productionScheduleGRPCHandler) ListProductionScheduleAtRiskOrders(ctx context.Context, req *pb.ListProductionScheduleAtRiskOrdersRequest) (*pb.ListProductionScheduleAtRiskOrdersResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	orders, apiErr := h.productionScheduleSvc.ListAtRiskOrders(ctx, req.ProductionScheduleId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	out := make([]*pb.ScheduleOrderCoverageProto, 0, len(orders))
	for _, o := range orders {
		covering := make([]*pb.ScheduleOrderCoverageLineProto, 0, len(o.CoveringLines))
		for _, l := range o.CoveringLines {
			covering = append(covering, &pb.ScheduleOrderCoverageLineProto{
				ProductionScheduleLineId: l.ProductionScheduleLineID,
				WeekIndex:                l.WeekIndex,
				MachineId:                l.MachineID,
				AllocatedQuantity:        l.AllocatedQuantity,
			})
		}
		row := &pb.ScheduleOrderCoverageProto{
			SalesOrderId:     o.SalesOrderID,
			SalesOrderNumber: o.SalesOrderNumber,
			ItemId:           o.ItemID,
			Sku:              o.SKU,
			UnitsAtRisk:      o.UnitsAtRisk,
			DueWeek:          safeconv.IntToInt32(o.DueWeek),
			ReasonCode:       o.ReasonCode,
			CoveringLines:    covering,
		}
		if o.ShipByDate != nil {
			row.ShipByDate = timestamppb.New(*o.ShipByDate)
		}
		out = append(out, row)
	}
	return &pb.ListProductionScheduleAtRiskOrdersResponse{Orders: out}, nil
}

func (h *productionScheduleGRPCHandler) QuotePromiseDate(ctx context.Context, req *pb.QuotePromiseDateRequest) (*pb.QuotePromiseDateResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	quote, apiErr := h.productionScheduleSvc.QuotePromiseDate(ctx, req.ItemId, req.Quantity)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	resp := &pb.QuotePromiseDateResponse{
		ItemId:                    quote.ItemID,
		Quantity:                  quote.Quantity,
		IsPromisable:              quote.IsPromisable,
		ProductionScheduleId:      quote.ProductionScheduleID,
		ProductionScheduleVersion: quote.ProductionScheduleVersion,
	}
	if quote.EarliestShipDate != nil {
		resp.EarliestShipDate = timestamppb.New(*quote.EarliestShipDate)
	}
	if quote.EarliestWeekIndex != nil {
		week := safeconv.IntToInt32(*quote.EarliestWeekIndex)
		resp.EarliestWeekIndex = &week
	}
	return resp, nil
}
