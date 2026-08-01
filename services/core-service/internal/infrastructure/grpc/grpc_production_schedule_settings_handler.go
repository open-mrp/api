package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"

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
