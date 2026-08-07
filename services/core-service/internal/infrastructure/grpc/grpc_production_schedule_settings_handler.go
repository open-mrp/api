package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/safeconv"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func settingsToProto(s *domain.ProductionScheduleSettings) *pb.ProductionScheduleSettingsInfo {
	info := &pb.ProductionScheduleSettingsInfo{
		PlanningHorizonWeeks:           s.PlanningHorizonWeeks,
		FrozenWeeks:                    s.FrozenWeeks,
		WeekStartDay:                   s.WeekStartDay,
		DemandWindowMonths:             s.DemandWindowMonths,
		ForecastHistoryMonths:          s.ForecastHistoryMonths,
		ForecastMonths:                 s.ForecastMonths,
		DemandBasisCode:                s.DemandBasisCode,
		ForecastZ:                      s.ForecastZ,
		ChangeoverAvgMinutes:           s.ChangeoverAvgMinutes,
		ChangeoverMinMinutes:           s.ChangeoverMinMinutes,
		ChangeoverMaxMinutes:           s.ChangeoverMaxMinutes,
		ChangeoverLaborRate:            s.ChangeoverLaborRate,
		HoldingRatePct:                 s.HoldingRatePct,
		ServiceLevelZ:                  s.ServiceLevelZ,
		FinishLeadTimeWeeks:            s.FinishLeadTimeWeeks,
		DefaultConstraintLeadTimeWeeks: s.DefaultConstraintLeadTimeWeeks,
		MaxWeeksSupply:                 s.MaxWeeksSupply,
		MaxFlowDepth:                   s.MaxFlowDepth,
		ShiftsPerDay:                   s.ShiftsPerDay,
		HoursPerShift:                  s.HoursPerShift,
		WorkDaysPerWeek:                s.WorkDaysPerWeek,
		WeeksPerYear:                   s.WeeksPerYear,
		CapacityHeadroomPct:            s.CapacityHeadroomPct,
		DefaultLotUnits:                s.DefaultLotUnits,
		DefaultCustomerLeadTimeDays:    s.DefaultCustomerLeadTimeDays,
		DefaultFulfillmentPolicyCode:   s.DefaultFulfillmentPolicyCode,
		IsEnabled:                      s.IsEnabled,
		GenerationTimezone:             s.GenerationTimezone,
		AutoPublish:                    s.AutoPublish,
		HasStoredSettings:              s.HasStoredSettings,
		CreatedAt:                      timestamppb.New(s.CreatedAt),
		UpdatedAt:                      timestamppb.New(s.UpdatedAt),
	}
	info.ConstraintDepartmentId = s.ConstraintDepartmentID
	info.GenerationCron = s.GenerationCron
	if s.LastGeneratedAt != nil {
		info.LastGeneratedAt = timestamppb.New(*s.LastGeneratedAt)
	}
	return info
}

func settingsFromProto(info *pb.ProductionScheduleSettingsInfo) domain.ProductionScheduleSettings {
	if info == nil {
		return domain.ProductionScheduleSettings{}
	}
	return domain.ProductionScheduleSettings{
		ConstraintDepartmentID:         info.ConstraintDepartmentId,
		PlanningHorizonWeeks:           info.PlanningHorizonWeeks,
		FrozenWeeks:                    info.FrozenWeeks,
		WeekStartDay:                   info.WeekStartDay,
		DemandWindowMonths:             info.DemandWindowMonths,
		ForecastHistoryMonths:          info.ForecastHistoryMonths,
		ForecastMonths:                 info.ForecastMonths,
		DemandBasisCode:                info.DemandBasisCode,
		ForecastZ:                      info.ForecastZ,
		ChangeoverAvgMinutes:           info.ChangeoverAvgMinutes,
		ChangeoverMinMinutes:           info.ChangeoverMinMinutes,
		ChangeoverMaxMinutes:           info.ChangeoverMaxMinutes,
		ChangeoverLaborRate:            info.ChangeoverLaborRate,
		HoldingRatePct:                 info.HoldingRatePct,
		ServiceLevelZ:                  info.ServiceLevelZ,
		FinishLeadTimeWeeks:            info.FinishLeadTimeWeeks,
		DefaultConstraintLeadTimeWeeks: info.DefaultConstraintLeadTimeWeeks,
		MaxWeeksSupply:                 info.MaxWeeksSupply,
		MaxFlowDepth:                   info.MaxFlowDepth,
		ShiftsPerDay:                   info.ShiftsPerDay,
		HoursPerShift:                  info.HoursPerShift,
		WorkDaysPerWeek:                info.WorkDaysPerWeek,
		WeeksPerYear:                   info.WeeksPerYear,
		CapacityHeadroomPct:            info.CapacityHeadroomPct,
		DefaultLotUnits:                info.DefaultLotUnits,
		DefaultCustomerLeadTimeDays:    info.DefaultCustomerLeadTimeDays,
		DefaultFulfillmentPolicyCode:   info.DefaultFulfillmentPolicyCode,
		IsEnabled:                      info.IsEnabled,
		GenerationCron:                 info.GenerationCron,
		GenerationTimezone:             info.GenerationTimezone,
		AutoPublish:                    info.AutoPublish,
	}
}

func (h *productionScheduleGRPCHandler) GetProductionScheduleSettings(ctx context.Context, req *pb.GetProductionScheduleSettingsRequest) (*pb.GetProductionScheduleSettingsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	settings, apiErr := h.productionScheduleSvc.GetProductionScheduleSettings(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.GetProductionScheduleSettingsResponse{Settings: settingsToProto(settings)}, nil
}

func (h *productionScheduleGRPCHandler) UpdateProductionScheduleSettings(ctx context.Context, req *pb.UpdateProductionScheduleSettingsRequest) (*pb.GetProductionScheduleSettingsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	settings, apiErr := h.productionScheduleSvc.UpdateProductionScheduleSettings(ctx, domain.UpdateProductionScheduleSettingsParams{
		Settings: settingsFromProto(req.Settings),
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.GetProductionScheduleSettingsResponse{Settings: settingsToProto(settings)}, nil
}

func resourceSettingToProto(s *domain.ProductionScheduleResourceSetting) *pb.ProductionScheduleResourceSettingInfo {
	info := &pb.ProductionScheduleResourceSettingInfo{
		Id:                  s.ID,
		ScopeCode:           s.ScopeCode,
		ScopeRefId:          s.ScopeRefID,
		IsExcluded:          s.IsExcluded,
		LeadTimeOffsetWeeks: s.LeadTimeOffsetWeeks,
	}
	info.LeadTimeWeeks = s.LeadTimeWeeks
	return info
}

func (h *productionScheduleGRPCHandler) ListProductionScheduleResourceSettings(ctx context.Context, req *pb.ListProductionScheduleResourceSettingsRequest) (*pb.ListProductionScheduleResourceSettingsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	settings, apiErr := h.productionScheduleSvc.ListResourceSettings(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	out := make([]*pb.ProductionScheduleResourceSettingInfo, len(settings))
	for i, setting := range settings {
		out[i] = resourceSettingToProto(setting)
	}
	return &pb.ListProductionScheduleResourceSettingsResponse{Settings: out}, nil
}

func (h *productionScheduleGRPCHandler) UpsertProductionScheduleResourceSetting(ctx context.Context, req *pb.UpsertProductionScheduleResourceSettingRequest) (*pb.UpsertProductionScheduleResourceSettingResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	setting, apiErr := h.productionScheduleSvc.UpsertResourceSetting(ctx, domain.UpsertResourceSettingParams{
		ScopeCode:           req.ScopeCode,
		ScopeRefID:          req.ScopeRefId,
		IsExcluded:          req.IsExcluded,
		LeadTimeWeeks:       req.LeadTimeWeeks,
		LeadTimeOffsetWeeks: req.LeadTimeOffsetWeeks,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.UpsertProductionScheduleResourceSettingResponse{Setting: resourceSettingToProto(setting)}, nil
}

func (h *productionScheduleGRPCHandler) DeleteProductionScheduleResourceSetting(ctx context.Context, req *pb.DeleteProductionScheduleResourceSettingRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	if apiErr := h.productionScheduleSvc.DeleteResourceSetting(ctx, req.Id); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &emptypb.Empty{}, nil
}

func itemSettingToProto(s *domain.ProductionScheduleItemPlanningSetting) *pb.ProductionScheduleItemSettingInfo {
	if s == nil {
		return nil
	}
	info := &pb.ProductionScheduleItemSettingInfo{
		Id:                    s.ID,
		ItemId:                s.ItemID,
		Sku:                   s.SKU,
		IsExcluded:            s.IsExcluded,
		LotMultipleUnits:      s.LotMultipleUnits,
		FulfillmentPolicyCode: s.FulfillmentPolicyCode,
		CreatedAt:             timestamppb.New(s.CreatedAt),
		UpdatedAt:             timestamppb.New(s.UpdatedAt),
	}
	return info
}

func (h *productionScheduleGRPCHandler) ListProductionScheduleItemSettings(ctx context.Context, req *pb.ListProductionScheduleItemSettingsRequest) (*pb.ListProductionScheduleItemSettingsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	settings, apiErr := h.productionScheduleSvc.ListItemSettings(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	out := make([]*pb.ProductionScheduleItemSettingInfo, 0, len(settings))
	for _, s := range settings {
		out = append(out, itemSettingToProto(s))
	}
	return &pb.ListProductionScheduleItemSettingsResponse{Settings: out}, nil
}

func (h *productionScheduleGRPCHandler) GetProductionScheduleItemSetting(ctx context.Context, req *pb.GetProductionScheduleItemSettingRequest) (*pb.ProductionScheduleItemSettingResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	setting, apiErr := h.productionScheduleSvc.GetItemSetting(ctx, req.ItemId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ProductionScheduleItemSettingResponse{Setting: itemSettingToProto(setting)}, nil
}

func (h *productionScheduleGRPCHandler) UpsertProductionScheduleItemSetting(ctx context.Context, req *pb.UpsertProductionScheduleItemSettingRequest) (*pb.ProductionScheduleItemSettingResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	setting, apiErr := h.productionScheduleSvc.UpsertItemSetting(ctx, domain.UpsertItemSettingParams{
		ItemID:                req.ItemId,
		IsExcluded:            req.IsExcluded,
		LotMultipleUnits:      req.LotMultipleUnits,
		FulfillmentPolicyCode: req.FulfillmentPolicyCode,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ProductionScheduleItemSettingResponse{Setting: itemSettingToProto(setting)}, nil
}

func (h *productionScheduleGRPCHandler) DeleteProductionScheduleItemSetting(ctx context.Context, req *pb.DeleteProductionScheduleItemSettingRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	if apiErr := h.productionScheduleSvc.DeleteItemSetting(ctx, req.ItemId); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &emptypb.Empty{}, nil
}

func fulfillmentRecommendationToProto(r *domain.FulfillmentRecommendation) *pb.FulfillmentRecommendationInfo {
	if r == nil {
		return nil
	}
	info := &pb.FulfillmentRecommendationInfo{
		ItemId:                     r.ItemID,
		Sku:                        r.SKU,
		CurrentPolicyCode:          r.CurrentPolicy,
		RecommendedPolicyCode:      r.RecommendedPolicy,
		ReasonCode:                 r.Reason,
		Changes:                    r.Changes(),
		AverageDemandInterval:      r.AverageDemandInterval,
		CoefficientOfVariation:     r.CoefficientOfVariation,
		TopCustomerSharePct:        r.TopCustomerSharePct,
		DemandWeightedLeadTimeDays: r.DemandWeightedLeadTimeDays,
		AnnualCogs:                 r.AnnualCOGS,
		MonthsSinceLastSale:        safeconv.IntToInt32(r.MonthsSinceLastSale),
		MixedStreamSharePct:        r.MixedStreamShare,
	}
	if r.ProductLineID != "" {
		info.ProductLineId = &r.ProductLineID
	}
	if r.TopCustomerName != "" {
		info.TopCustomerName = &r.TopCustomerName
	}
	return info
}

func (h *productionScheduleGRPCHandler) ListFulfillmentRecommendations(ctx context.Context, req *pb.ListFulfillmentRecommendationsRequest) (*pb.ListFulfillmentRecommendationsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	recommendations, apiErr := h.productionScheduleSvc.ListFulfillmentRecommendations(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	out := make([]*pb.FulfillmentRecommendationInfo, 0, len(recommendations))
	for _, r := range recommendations {
		out = append(out, fulfillmentRecommendationToProto(r))
	}
	return &pb.ListFulfillmentRecommendationsResponse{Recommendations: out}, nil
}

func (h *productionScheduleGRPCHandler) ApplyFulfillmentRecommendations(ctx context.Context, req *pb.ApplyFulfillmentRecommendationsRequest) (*pb.ListFulfillmentRecommendationsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	applied, apiErr := h.productionScheduleSvc.ApplyFulfillmentRecommendations(ctx, req.ItemIds)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	out := make([]*pb.FulfillmentRecommendationInfo, 0, len(applied))
	for _, r := range applied {
		out = append(out, fulfillmentRecommendationToProto(r))
	}
	return &pb.ListFulfillmentRecommendationsResponse{Recommendations: out}, nil
}
