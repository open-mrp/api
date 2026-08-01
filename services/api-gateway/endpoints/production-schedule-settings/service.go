package productionschedulesettingsep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProductionScheduleSettingsSvc interface {
	GetSettings(ctx context.Context, req *RetrieveProductionScheduleSettingsRequest) (*apiresource.ProductionScheduleSettings, *apierror.APIError)
	UpdateSettings(ctx context.Context, req *UpdateProductionScheduleSettingsRequest) (*apiresource.ProductionScheduleSettings, *apierror.APIError)
	ListResourceSettings(ctx context.Context, req *ListResourceSettingsRequest) (*apiresource.List[apiresource.ProductionScheduleResourceSetting], *apierror.APIError)
	UpsertResourceSetting(ctx context.Context, req *UpsertResourceSettingRequest) (*apiresource.ProductionScheduleResourceSetting, *apierror.APIError)
	DeleteResourceSetting(ctx context.Context, req *DeleteResourceSettingRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type ProductionScheduleSettingsSvcConfig struct {
	// CoreClient (required) is the core-service production-schedule gRPC client.
	CoreClient pb.CoreProductionScheduleServiceClient
}

type settingsSvcImpl struct {
	coreClient pb.CoreProductionScheduleServiceClient
}

var settingsEpSvcTracer = tracing.GetTracer("api-gateway.endpoints.production-schedule-settings.service")

func (c *ProductionScheduleSettingsSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("production schedule settings endpoint service: core client is required")
	}
	return nil
}

func NewProductionScheduleSettingsSvc(config *ProductionScheduleSettingsSvcConfig) ProductionScheduleSettingsSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &settingsSvcImpl{coreClient: config.CoreClient}
}

func settingsFromProto(info *pb.ProductionScheduleSettingsInfo) apiresource.ProductionScheduleSettings {
	out := apiresource.ProductionScheduleSettings{
		Object:                         constants.ObjectTypeProductionScheduleSettings,
		PlanningHorizonWeeks:           info.PlanningHorizonWeeks,
		FrozenWeeks:                    info.FrozenWeeks,
		WeekStartDay:                   info.WeekStartDay,
		DemandWindowMonths:             info.DemandWindowMonths,
		ForecastHistoryMonths:          info.ForecastHistoryMonths,
		ForecastMonths:                 info.ForecastMonths,
		DemandBasis:                    constants.ScheduleDemandBasis(info.DemandBasisCode),
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
		CadenceStatus:                  constants.ActivationStatusOf(info.IsEnabled),
		GenerationCron:                 info.GenerationCron,
		GenerationTimezone:             info.GenerationTimezone,
		AutoPublishStatus:              constants.ActivationStatusOf(info.AutoPublish),
		SettingsStatus:                 constants.SettingsStatusOf(info.HasStoredSettings),
		CreatedAt:                      grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:                      grpcutil.TimestampToTime(info.UpdatedAt),
	}
	out.LastGeneratedAt = grpcutil.TimestampToTimePtr(info.LastGeneratedAt)
	if info.ConstraintDepartmentId != nil && *info.ConstraintDepartmentId != "" {
		out.ConstraintDepartment = apiresource.NewEntity(*info.ConstraintDepartmentId, constants.ObjectTypeDepartment, nil, nil)
	}
	return out
}

func (m *settingsSvcImpl) GetSettings(ctx context.Context, req *RetrieveProductionScheduleSettingsRequest) (*apiresource.ProductionScheduleSettings, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, settingsEpSvcTracer, "service.production_schedule_settings.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetProductionScheduleSettingsResponse, error) {
			return m.coreClient.GetProductionScheduleSettings(ctx, &pb.GetProductionScheduleSettingsRequest{}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	settings := settingsFromProto(resp.Settings)
	return &settings, nil
}

func (m *settingsSvcImpl) UpdateSettings(ctx context.Context, req *UpdateProductionScheduleSettingsRequest) (*apiresource.ProductionScheduleSettings, *apierror.APIError) {
	pbReq := &pb.UpdateProductionScheduleSettingsRequest{
		Settings: &pb.ProductionScheduleSettingsInfo{
			ConstraintDepartmentId:         req.ConstraintDepartmentID.ValuePtr(),
			PlanningHorizonWeeks:           req.PlanningHorizonWeeks,
			FrozenWeeks:                    req.FrozenWeeks,
			WeekStartDay:                   req.WeekStartDay,
			DemandWindowMonths:             req.DemandWindowMonths,
			ForecastHistoryMonths:          req.ForecastHistoryMonths,
			ForecastMonths:                 req.ForecastMonths,
			DemandBasisCode:                string(req.DemandBasis),
			ForecastZ:                      req.ForecastZ,
			ChangeoverAvgMinutes:           req.ChangeoverAvgMinutes,
			ChangeoverMinMinutes:           req.ChangeoverMinMinutes,
			ChangeoverMaxMinutes:           req.ChangeoverMaxMinutes,
			ChangeoverLaborRate:            req.ChangeoverLaborRate,
			HoldingRatePct:                 req.HoldingRatePct,
			ServiceLevelZ:                  req.ServiceLevelZ,
			FinishLeadTimeWeeks:            req.FinishLeadTimeWeeks,
			DefaultConstraintLeadTimeWeeks: req.DefaultConstraintLeadTimeWeeks,
			MaxWeeksSupply:                 req.MaxWeeksSupply,
			MaxFlowDepth:                   req.MaxFlowDepth,
			ShiftsPerDay:                   req.ShiftsPerDay,
			HoursPerShift:                  req.HoursPerShift,
			WorkDaysPerWeek:                req.WorkDaysPerWeek,
			WeeksPerYear:                   req.WeeksPerYear,
			CapacityHeadroomPct:            req.CapacityHeadroomPct,
			DefaultLotUnits:                req.DefaultLotUnits,
			IsEnabled:                      req.CadenceStatus == constants.ActivationStatusActive,
			GenerationCron:                 req.GenerationCron.ValuePtr(),
			GenerationTimezone:             req.GenerationTimezone,
			AutoPublish:                    req.AutoPublishStatus == constants.ActivationStatusActive,
		},
	}

	resp, apiErr := grpcutil.CallRPC(ctx, settingsEpSvcTracer, "service.production_schedule_settings.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetProductionScheduleSettingsResponse, error) {
			return m.coreClient.UpdateProductionScheduleSettings(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	settings := settingsFromProto(resp.Settings)
	return &settings, nil
}

func resourceSettingFromProto(info *pb.ProductionScheduleResourceSettingInfo) apiresource.ProductionScheduleResourceSetting {
	out := apiresource.ProductionScheduleResourceSetting{
		ID:                  info.Id,
		Object:              constants.ObjectTypeProductionScheduleResourceSetting,
		ScopeType:           constants.ScheduleResourceScope(info.ScopeCode),
		Scope:               resourceSettingScopeEntity(info),
		ParticipationStatus: constants.ParticipationStatusOf(info.IsExcluded),
		LeadTimeOffsetWeeks: info.LeadTimeOffsetWeeks,
	}
	out.LeadTimeWeeks = info.LeadTimeWeeks
	return out
}

// resourceSettingScopeEntity maps the scope code onto the object type it references.
func resourceSettingScopeEntity(info *pb.ProductionScheduleResourceSettingInfo) *apiresource.Entity {
	if info.ScopeRefId == "" {
		return nil
	}
	var scopeType constants.ObjectType
	switch constants.ScheduleResourceScope(info.ScopeCode) {
	case constants.ScheduleResourceScopeMachine:
		scopeType = constants.ObjectTypeMachine
	case constants.ScheduleResourceScopeDepartment:
		scopeType = constants.ObjectTypeDepartment
	case constants.ScheduleResourceScopeProductionStep:
		scopeType = constants.ObjectTypeProductionStep
	default:
		return nil
	}
	return apiresource.NewEntity(info.ScopeRefId, scopeType, nil, nil)
}

func (m *settingsSvcImpl) ListResourceSettings(ctx context.Context, req *ListResourceSettingsRequest) (*apiresource.List[apiresource.ProductionScheduleResourceSetting], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, settingsEpSvcTracer, "service.production_schedule_settings.list_resources", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListProductionScheduleResourceSettingsResponse, error) {
			return m.coreClient.ListProductionScheduleResourceSettings(ctx, &pb.ListProductionScheduleResourceSettingsRequest{}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	settings := make([]apiresource.ProductionScheduleResourceSetting, len(resp.Settings))
	for i, s := range resp.Settings {
		settings[i] = resourceSettingFromProto(s)
	}
	return apiresource.NewList(settings, apiresource.PageInfo{}), nil
}

func (m *settingsSvcImpl) UpsertResourceSetting(ctx context.Context, req *UpsertResourceSettingRequest) (*apiresource.ProductionScheduleResourceSetting, *apierror.APIError) {
	pbReq := &pb.UpsertProductionScheduleResourceSettingRequest{
		ScopeCode:           string(req.ScopeType),
		ScopeRefId:          req.ScopeRefID,
		IsExcluded:          req.ParticipationStatus == constants.ParticipationStatusExcluded,
		LeadTimeWeeks:       req.LeadTimeWeeks.Ptr(),
		LeadTimeOffsetWeeks: req.LeadTimeOffsetWeeks,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, settingsEpSvcTracer, "service.production_schedule_settings.upsert_resource", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpsertProductionScheduleResourceSettingResponse, error) {
			return m.coreClient.UpsertProductionScheduleResourceSetting(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	setting := resourceSettingFromProto(resp.Setting)
	return &setting, nil
}

func (m *settingsSvcImpl) DeleteResourceSetting(ctx context.Context, req *DeleteResourceSettingRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, settingsEpSvcTracer, "service.production_schedule_settings.delete_resource", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteProductionScheduleResourceSetting(ctx, &pb.DeleteProductionScheduleResourceSettingRequest{Id: req.SettingID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}
