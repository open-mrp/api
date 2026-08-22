package productionschedulesettingsep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProductionScheduleSettingsSvc interface {
	GetSettings(ctx context.Context, req *RetrieveProductionScheduleSettingsRequest) (*apiresource.ProductionScheduleSettings, *apierror.APIError)
	UpdateSettings(ctx context.Context, req *UpdateProductionScheduleSettingsRequest) (*apiresource.ProductionScheduleSettings, *apierror.APIError)
	ListResourceSettings(ctx context.Context, req *ListResourceSettingsRequest) (*apiresource.List[apiresource.ProductionScheduleResourceSetting], *apierror.APIError)
	UpsertResourceSetting(ctx context.Context, req *UpsertResourceSettingRequest) (*apiresource.ProductionScheduleResourceSetting, *apierror.APIError)
	DeleteResourceSetting(ctx context.Context, req *DeleteResourceSettingRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ListItemSettings(ctx context.Context, req *ListItemSettingsRequest) (*apiresource.List[apiresource.ProductionScheduleItemSetting], *apierror.APIError)
	GetItemSetting(ctx context.Context, req *RetrieveItemSettingRequest) (*apiresource.ProductionScheduleItemSetting, *apierror.APIError)
	UpsertItemSetting(ctx context.Context, req *UpsertItemSettingRequest) (*apiresource.ProductionScheduleItemSetting, *apierror.APIError)
	DeleteItemSetting(ctx context.Context, req *DeleteItemSettingRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ListFulfillmentRecommendations(ctx context.Context, req *ListFulfillmentRecommendationsRequest) (*apiresource.List[apiresource.FulfillmentRecommendation], *apierror.APIError)
	ApplyFulfillmentRecommendations(ctx context.Context, req *ApplyFulfillmentRecommendationsRequest) (*apiresource.List[apiresource.FulfillmentRecommendation], *apierror.APIError)
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
		DefaultCustomerLeadTimeDays:    info.DefaultCustomerLeadTimeDays,
		ShipCalendarID:                 info.ShipCalendarId,
		ReceiveCalendarID:              info.ReceiveCalendarId,
		DefaultFulfillmentPolicy:       constants.FulfillmentPolicy(info.DefaultFulfillmentPolicyCode),
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
			DefaultCustomerLeadTimeDays:    req.DefaultCustomerLeadTimeDays,
			ShipCalendarId:                 req.ShipCalendarID.ValuePtr(),
			ReceiveCalendarId:              req.ReceiveCalendarID.ValuePtr(),
			DefaultFulfillmentPolicyCode:   string(req.DefaultFulfillmentPolicy),
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

// itemSettingFromProto maps one item's planning overrides onto the API shape.
func itemSettingFromProto(s *pb.ProductionScheduleItemSettingInfo) *apiresource.ProductionScheduleItemSetting {
	if s == nil {
		return nil
	}
	participation := constants.ParticipationStatusIncluded
	if s.IsExcluded {
		participation = constants.ParticipationStatusExcluded
	}
	out := &apiresource.ProductionScheduleItemSetting{
		ID:                  s.Id,
		Object:              constants.ObjectTypeProductionScheduleItemSetting,
		Item:                apiresource.NewEntity(s.ItemId, constants.ObjectTypeItem, nil, &s.Sku),
		ParticipationStatus: participation,
		LotMultipleUnits:    s.LotMultipleUnits,
		CreatedAt:           grpcutil.TimestampToTime(s.CreatedAt),
		UpdatedAt:           grpcutil.TimestampToTime(s.UpdatedAt),
	}
	if s.FulfillmentPolicyCode != nil && *s.FulfillmentPolicyCode != "" {
		policy := constants.FulfillmentPolicy(*s.FulfillmentPolicyCode)
		out.FulfillmentPolicy = &policy
	}
	return out
}

func (m *settingsSvcImpl) ListItemSettings(ctx context.Context, _ *ListItemSettingsRequest) (*apiresource.List[apiresource.ProductionScheduleItemSetting], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, settingsEpSvcTracer, "service.production_schedule_settings.list_item_settings", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListProductionScheduleItemSettingsResponse, error) {
			return m.coreClient.ListProductionScheduleItemSettings(ctx, &pb.ListProductionScheduleItemSettingsRequest{}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	items := make([]apiresource.ProductionScheduleItemSetting, 0, len(resp.Settings))
	for _, s := range resp.Settings {
		if mapped := itemSettingFromProto(s); mapped != nil {
			items = append(items, *mapped)
		}
	}
	return apiresource.NewList(items, apiresource.PageInfo{}), nil
}

func (m *settingsSvcImpl) UpsertItemSetting(ctx context.Context, req *UpsertItemSettingRequest) (*apiresource.ProductionScheduleItemSetting, *apierror.APIError) {
	pbReq := &pb.UpsertProductionScheduleItemSettingRequest{
		ItemId:     req.ItemID,
		IsExcluded: req.ParticipationStatus == constants.ParticipationStatusExcluded,
	}
	if v, ok := req.LotMultipleUnits.Value(); ok {
		pbReq.LotMultipleUnits = &v
	}
	// A clear sends nothing, which is what returns the item to its product line's policy.
	if v, ok := req.FulfillmentPolicy.Value(); ok {
		policy := string(v)
		pbReq.FulfillmentPolicyCode = &policy
	}

	resp, apiErr := grpcutil.CallRPC(ctx, settingsEpSvcTracer, "service.production_schedule_settings.upsert_item_setting", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ProductionScheduleItemSettingResponse, error) {
			return m.coreClient.UpsertProductionScheduleItemSetting(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return itemSettingFromProto(resp.Setting), nil
}

func (m *settingsSvcImpl) DeleteItemSetting(ctx context.Context, req *DeleteItemSettingRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, settingsEpSvcTracer, "service.production_schedule_settings.delete_item_setting", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteProductionScheduleItemSetting(ctx, &pb.DeleteProductionScheduleItemSettingRequest{ItemId: req.ItemID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

// recommendationFromProto maps one item's fulfillment advice onto the API shape.
func recommendationFromProto(r *pb.FulfillmentRecommendationInfo) apiresource.FulfillmentRecommendation {
	out := apiresource.FulfillmentRecommendation{
		Object:                     constants.ObjectTypeFulfillmentRecommendation,
		Item:                       apiresource.NewEntity(r.ItemId, constants.ObjectTypeItem, r.Description, &r.Sku),
		SKU:                        r.Sku,
		CurrentPolicy:              constants.FulfillmentPolicy(r.CurrentPolicyCode),
		RecommendedPolicy:          constants.FulfillmentPolicy(r.RecommendedPolicyCode),
		Changes:                    r.Changes,
		Reason:                     constants.FulfillmentRecommendationReason(r.ReasonCode),
		AverageDemandInterval:      r.AverageDemandInterval,
		CoefficientOfVariation:     r.CoefficientOfVariation,
		TopCustomerSharePct:        r.TopCustomerSharePct,
		TopCustomerName:            r.TopCustomerName,
		DemandWeightedLeadTimeDays: r.DemandWeightedLeadTimeDays,
		AnnualCOGS:                 r.AnnualCogs,
		MonthsSinceLastSale:        r.MonthsSinceLastSale,
		MixedStreamSharePct:        r.MixedStreamSharePct,
	}
	if r.ProductLineId != nil {
		out.ProductLine = apiresource.NewEntity(*r.ProductLineId, constants.ObjectTypeProductLine, nil, nil)
	}
	return out
}

func (m *settingsSvcImpl) ListFulfillmentRecommendations(ctx context.Context, _ *ListFulfillmentRecommendationsRequest) (*apiresource.List[apiresource.FulfillmentRecommendation], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, settingsEpSvcTracer, "service.production_schedule_settings.list_fulfillment_recommendations", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListFulfillmentRecommendationsResponse, error) {
			return m.coreClient.ListFulfillmentRecommendations(ctx, &pb.ListFulfillmentRecommendationsRequest{}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	items := make([]apiresource.FulfillmentRecommendation, 0, len(resp.Recommendations))
	for _, r := range resp.Recommendations {
		items = append(items, recommendationFromProto(r))
	}
	return apiresource.NewList(items, apiresource.PageInfo{}), nil
}

func (m *settingsSvcImpl) ApplyFulfillmentRecommendations(ctx context.Context, req *ApplyFulfillmentRecommendationsRequest) (*apiresource.List[apiresource.FulfillmentRecommendation], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, settingsEpSvcTracer, "service.production_schedule_settings.apply_fulfillment_recommendations", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListFulfillmentRecommendationsResponse, error) {
			return m.coreClient.ApplyFulfillmentRecommendations(ctx, &pb.ApplyFulfillmentRecommendationsRequest{ItemIds: req.ItemIDs}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	items := make([]apiresource.FulfillmentRecommendation, 0, len(resp.Recommendations))
	for _, r := range resp.Recommendations {
		items = append(items, recommendationFromProto(r))
	}
	return apiresource.NewList(items, apiresource.PageInfo{}), nil
}

func (m *settingsSvcImpl) GetItemSetting(ctx context.Context, req *RetrieveItemSettingRequest) (*apiresource.ProductionScheduleItemSetting, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, settingsEpSvcTracer, "service.production_schedule_settings.get_item_setting", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ProductionScheduleItemSettingResponse, error) {
			return m.coreClient.GetProductionScheduleItemSetting(ctx, &pb.GetProductionScheduleItemSettingRequest{ItemId: req.ItemID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return itemSettingFromProto(resp.Setting), nil
}
