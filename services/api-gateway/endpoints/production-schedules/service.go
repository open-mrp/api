package productionscheduleep

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	productionrunep "github.com/augno/api/services/api-gateway/endpoints/production-runs"
	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProductionScheduleSvc interface {
	PreviewProductionSchedule(ctx context.Context, req *PreviewProductionScheduleRequest) (*apiresource.ProductionSchedulePreview, *apierror.APIError)
	GenerateProductionSchedule(ctx context.Context, req *GenerateProductionScheduleRequest) (*apiresource.ProductionSchedule, *apierror.APIError)
	PreviewRegenerateProductionSchedule(ctx context.Context, req *PreviewRegenerateProductionScheduleRequest) (*apiresource.ProductionScheduleRegeneratePreview, *apierror.APIError)
	RegenerateProductionSchedule(ctx context.Context, req *RegenerateProductionScheduleRequest) (*apiresource.ProductionSchedule, *apierror.APIError)
	GetProductionSchedule(ctx context.Context, req *RetrieveProductionScheduleRequest) (*apiresource.ProductionSchedule, *apierror.APIError)
	GetCurrentProductionSchedule(ctx context.Context, req *RetrieveCurrentProductionScheduleRequest) (*apiresource.ProductionSchedule, *apierror.APIError)
	ListProductionSchedules(ctx context.Context, req *ListProductionSchedulesRequest) (*apiresource.List[apiresource.ProductionSchedule], *apierror.APIError)
	ListProductionScheduleLines(ctx context.Context, req *ListProductionScheduleLinesRequest) (*apiresource.List[apiresource.ProductionScheduleLine], *apierror.APIError)
	ListProductionScheduleItemPolicies(ctx context.Context, req *ListProductionScheduleItemPoliciesRequest) (*apiresource.List[apiresource.ProductionScheduleItemPolicy], *apierror.APIError)
	ListProductionScheduleFinishedPolicies(ctx context.Context, req *ListProductionScheduleFinishedPoliciesRequest) (*apiresource.List[apiresource.ProductionScheduleFinishedPolicy], *apierror.APIError)
	ListProductionScheduleFinishingLines(ctx context.Context, req *ListProductionScheduleFinishingLinesRequest) (*apiresource.List[apiresource.ProductionScheduleFinishingLine], *apierror.APIError)
	ListProductionScheduleDerivedLines(ctx context.Context, req *ListProductionScheduleDerivedLinesRequest) (*apiresource.List[apiresource.ProductionScheduleDerivedLine], *apierror.APIError)
	ListAtRiskOrders(ctx context.Context, req *ListAtRiskOrdersRequest) (*apiresource.List[apiresource.ScheduleOrderCoverage], *apierror.APIError)
	QuotePromiseDate(ctx context.Context, req *QuotePromiseDateRequest) (*apiresource.PromiseDateQuote, *apierror.APIError)
	ListScheduleDeviationTypes(ctx context.Context, req *ListScheduleDeviationTypesRequest) (*apiresource.List[apiresource.ScheduleDeviationType], *apierror.APIError)
	ListProductionScheduleDeviations(ctx context.Context, req *ListProductionScheduleDeviationsRequest) (*apiresource.List[apiresource.ProductionScheduleDeviation], *apierror.APIError)
	CreateProductionScheduleLine(ctx context.Context, req *CreateProductionScheduleLineRequest) (*apiresource.ProductionScheduleLine, *apierror.APIError)
	UpdateProductionScheduleLine(ctx context.Context, req *UpdateProductionScheduleLineRequest) (*apiresource.ProductionScheduleLine, *apierror.APIError)
	DeleteProductionScheduleLine(ctx context.Context, req *DeleteProductionScheduleLineRequest) (*apiresource.EmptyResource, *apierror.APIError)
	PublishProductionSchedule(ctx context.Context, req *PublishProductionScheduleRequest) (*apiresource.ProductionSchedule, *apierror.APIError)
	ReleaseProductionScheduleWeek(ctx context.Context, req *ReleaseProductionScheduleWeekRequest) (*apiresource.ReleaseScheduleWeekResult, *apierror.APIError)
	PreviewReleaseProductionScheduleWeek(ctx context.Context, req *PreviewReleaseProductionScheduleWeekRequest) (*apiresource.ReleaseScheduleWeekPreview, *apierror.APIError)
	ArchiveProductionSchedule(ctx context.Context, req *ArchiveProductionScheduleRequest) (*apiresource.ProductionSchedule, *apierror.APIError)
	DeleteProductionSchedule(ctx context.Context, req *DeleteProductionScheduleRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type ProductionScheduleSvcConfig struct {
	// CoreClient (required) is the core-service production-schedule gRPC client.
	CoreClient pb.CoreProductionScheduleServiceClient
}

type productionScheduleSvcImpl struct {
	coreClient pb.CoreProductionScheduleServiceClient
}

var productionScheduleEpSvcTracer = tracing.GetTracer("api-gateway.endpoints.production-schedules.service")

func (c *ProductionScheduleSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("production schedule endpoint service: core client is required")
	}
	return nil
}

func NewProductionScheduleSvc(config *ProductionScheduleSvcConfig) ProductionScheduleSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &productionScheduleSvcImpl{coreClient: config.CoreClient}
}

func (m *productionScheduleSvcImpl) PreviewProductionSchedule(ctx context.Context, req *PreviewProductionScheduleRequest) (*apiresource.ProductionSchedulePreview, *apierror.APIError) {
	pbReq := &pb.PreviewProductionScheduleRequest{}
	if planningAsOf := req.PlanningAsOf.Ptr(); planningAsOf != nil {
		pbReq.PlanningAsOf = timestamppb.New(*planningAsOf)
	}
	if horizonWeeks := req.HorizonWeeks.Ptr(); horizonWeeks != nil {
		pbReq.HorizonWeeks = horizonWeeks
	}
	if demandBasis := req.DemandBasis.Ptr(); demandBasis != nil {
		pbReq.DemandBasis = scheduleEnumPtr(demandBasis)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedules.preview", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.PreviewProductionScheduleResponse, error) {
			return m.coreClient.PreviewProductionSchedule(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return previewFromProto(resp), nil
}

// entityRef builds a light {id, object} reference to another resource, or nil when the id is absent so the field serializes as null.
func entityRef(id string, objectType constants.ObjectType) *apiresource.Entity {
	if id == "" {
		return nil
	}
	return apiresource.NewEntity(id, objectType, nil, nil)
}

// entityRefPtr is entityRef for nullable foreign keys.
func entityRefPtr(id *string, objectType constants.ObjectType) *apiresource.Entity {
	if id == nil {
		return nil
	}
	return entityRef(*id, objectType)
}

func previewFromProto(resp *pb.PreviewProductionScheduleResponse) *apiresource.ProductionSchedulePreview {
	policies := make([]apiresource.SchedulePolicy, len(resp.Policies))
	campaigns := make([]apiresource.ScheduleCampaign, len(resp.Campaigns))
	projections := make([]apiresource.ScheduleProjection, len(resp.Projections))
	out := &apiresource.ProductionSchedulePreview{
		Object:        constants.ObjectTypeProductionSchedulePreview,
		SolverVersion: resp.SolverVersion,
		PlanningAsOf:  grpcutil.TimestampToTime(resp.PlanningAsOf),
	}

	for i, p := range resp.Policies {
		policies[i] = apiresource.SchedulePolicy{
			Item:                    entityRef(p.ItemId, constants.ObjectTypeItem),
			SKU:                     p.Sku,
			AnnualDemand:            p.AnnualDemand,
			WeeklyDemand:            p.WeeklyDemand,
			SecondsPerUnit:          p.SecondsPerUnit,
			UnitCost:                p.UnitCost,
			SetupCost:               p.SetupCost,
			HoldingCost:             p.HoldingCost,
			EOQUnits:                p.EoqUnits,
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
			ABCClass:                abcClassPtr(p.AbcClass),
			FulfillmentPolicy:       constants.FulfillmentPolicy(p.FulfillmentPolicyCode),
			PolicySource:            constants.FulfillmentPolicySource(p.PolicySourceCode),
			FirmDemandUnits:         p.FirmDemandUnits,
			ForecastDemandUnits:     p.ForecastDemandUnits,
			AnnualRunHours:          p.AnnualRunHours,
		}
	}

	for i, c := range resp.Campaigns {
		campaigns[i] = apiresource.ScheduleCampaign{
			Item:      entityRef(c.ItemId, constants.ObjectTypeItem),
			SKU:       c.Sku,
			Machine:   entityRef(c.MachineId, constants.ObjectTypeMachine),
			WeekIndex: c.WeekIndex,
			Units:     c.Units,
			Lots:      c.Lots,
			RunHours:  c.RunHours,
		}
	}

	for i, p := range resp.Projections {
		projections[i] = apiresource.ScheduleProjection{
			Item:         entityRef(p.ItemId, constants.ObjectTypeItem),
			OnHandByWeek: p.OnHandByWeek,
		}
	}

	out.Policies = apiresource.NewList(policies, apiresource.PageInfo{})
	out.Campaigns = apiresource.NewList(campaigns, apiresource.PageInfo{})
	out.Projections = apiresource.NewList(projections, apiresource.PageInfo{})

	// Slices are initialized rather than left nil so they serialize as [] and a client never has to distinguish "no diagnostics" from "field missing".
	diagnostics := emptyScheduleDiagnostics()
	if d := resp.Diagnostics; d != nil {
		if len(d.EoqCappedSkus) > 0 {
			diagnostics.EOQCappedSKUs = d.EoqCappedSkus
		}
		if len(d.UnschedulableSkus) > 0 {
			diagnostics.UnschedulableSKUs = d.UnschedulableSkus
		}
		if len(d.CapacityStarvedSkus) > 0 {
			diagnostics.CapacityStarvedSKUs = d.CapacityStarvedSkus
		}
		if len(d.ItemsWithoutRunRate) > 0 {
			diagnostics.ItemsWithoutRunRate = d.ItemsWithoutRunRate
		}
		diagnostics.ExcludedItemCount = d.ExcludedItemCount
		diagnostics.ConstraintMachineCount = d.ConstraintMachineCount
		diagnostics.MeasuredBatchCount = d.MeasuredBatchCount
		diagnostics.MachinesWithoutStep = d.MachinesWithoutStep
		diagnostics.ChangeoverSlopeMinutes = d.ChangeoverSlopeMinutes
		diagnostics.AverageInputsAdded = d.AverageInputsAdded
		diagnostics.FirmDemandUnits = d.FirmDemandUnits
		diagnostics.UndatedFirmOrderCount = d.UndatedFirmOrderCount
		diagnostics.MakeToOrderItemCount = d.MakeToOrderItemCount
		diagnostics.FinishingMachineCount = d.FinishingMachineCount
		diagnostics.FinishingCapacityIsEstimated = d.FinishingCapacityIsEstimated
		if f := d.Finishing; f != nil {
			diagnostics.Finishing = apiresource.ScheduleFinishingDiagnostics{
				WeeklyCapacityHours: f.WeeklyCapacityHours,
				PlannedHoursByWeek:  nonNilFloats(f.PlannedHoursByWeek),
				UtilisationByWeek:   nonNilFloats(f.UtilisationByWeek),
				GreigeStarvedSKUs:   nonNilStrings(f.GreigeStarvedSkus),
				CapacityStarvedSKUs: nonNilStrings(f.CapacityStarvedSkus),
				ItemsWithoutRunRate: nonNilStrings(f.ItemsWithoutRunRate),
				UnusedGreigeUnits:   f.UnusedGreigeUnits,
				TotalPlannedUnits:   f.TotalPlannedUnits,
				LineCount:           f.LineCount,
			}
		}

		for _, o := range d.AtRiskOrders {
			order := apiresource.ScheduleAtRiskOrder{
				Object:  constants.ObjectTypeScheduleAtRiskOrder,
				SKU:     o.Sku,
				Units:   o.Units,
				DueWeek: o.DueWeek,
				Reason:  constants.ScheduleAtRiskReason(o.Reason),
			}
			order.SalesOrder = entityRef(o.SalesOrderId, constants.ObjectTypeSalesOrder)
			if o.SalesOrderNumber != "" && order.SalesOrder != nil {
				order.SalesOrder = apiresource.NewEntity(o.SalesOrderId, constants.ObjectTypeSalesOrder, nil, &o.SalesOrderNumber)
			}
			if o.ItemId != "" {
				label := o.Sku
				order.Item = apiresource.NewEntity(o.ItemId, constants.ObjectTypeItem, nil, &label)
			}
			diagnostics.AtRiskOrders.Data = append(diagnostics.AtRiskOrders.Data, order)
		}

		for _, o := range d.AppliedOverrides {
			diagnostics.AppliedOverrides.Data = append(diagnostics.AppliedOverrides.Data, apiresource.ScheduleAppliedOverride{
				Override:   entityRef(o.OverrideId, constants.ObjectTypeDemandOverride),
				Item:       entityRef(o.ItemId, constants.ObjectTypeItem),
				MonthStart: grpcutil.TimestampToTime(o.MonthStart),
				Before:     o.Before,
				After:      o.After,
				Adjustment: constants.DemandOverrideAdjustment(o.TypeCode),
				Reason:     constants.DemandOverrideReasonPtr(nonEmptyPtr(o.ReasonCode)),
			})
		}
	}
	out.Diagnostics = diagnostics

	return out
}

func (m *productionScheduleSvcImpl) GenerateProductionSchedule(ctx context.Context, req *GenerateProductionScheduleRequest) (*apiresource.ProductionSchedule, *apierror.APIError) {
	pbReq := &pb.GenerateProductionScheduleRequest{
		HorizonWeeks: req.HorizonWeeks.Ptr(),
		DemandBasis:  scheduleEnumPtr(req.DemandBasis.Ptr()),
		Name:         req.Name.Ptr(),
	}
	if planningAsOf := req.PlanningAsOf.Ptr(); planningAsOf != nil {
		pbReq.PlanningAsOf = timestamppb.New(*planningAsOf)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedules.generate", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GenerateProductionScheduleResponse, error) {
			return m.coreClient.GenerateProductionSchedule(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return scheduleFromProto(resp.Schedule), nil
}

func (m *productionScheduleSvcImpl) GetProductionSchedule(ctx context.Context, req *RetrieveProductionScheduleRequest) (*apiresource.ProductionSchedule, *apierror.APIError) {
	pbReq := &pb.GetProductionScheduleRequest{Id: req.ProductionScheduleID}

	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedules.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetProductionScheduleResponse, error) {
			return m.coreClient.GetProductionSchedule(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return scheduleFromProto(resp.Schedule), nil
}

func (m *productionScheduleSvcImpl) GetCurrentProductionSchedule(ctx context.Context, req *RetrieveCurrentProductionScheduleRequest) (*apiresource.ProductionSchedule, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedules.get_current", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetProductionScheduleResponse, error) {
			return m.coreClient.GetCurrentProductionSchedule(ctx, &pb.GetCurrentProductionScheduleRequest{}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	// No published version covering today is a normal state, but the caller asked for "the current schedule" — a 404 says that honestly, where an empty object would look like a real but blank plan.
	if resp.Schedule == nil {
		return nil, apierror.NewResourceNotFoundError("No published production schedule covers today.")
	}
	return scheduleFromProto(resp.Schedule), nil
}

func (m *productionScheduleSvcImpl) ListProductionSchedules(ctx context.Context, req *ListProductionSchedulesRequest) (*apiresource.List[apiresource.ProductionSchedule], *apierror.APIError) {
	pbReq := &pb.ListProductionSchedulesRequest{
		Cursor:      req.Cursor,
		Limit:       req.Limit,
		StatusCodes: enumStrings(req.Statuses),
		Query:       req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedules.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListProductionSchedulesResponse, error) {
			return m.coreClient.ListProductionSchedules(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	schedules := make([]apiresource.ProductionSchedule, len(resp.Schedules))
	for i, s := range resp.Schedules {
		schedules[i] = *scheduleFromProto(s)
	}
	return apiresource.NewList(schedules, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *productionScheduleSvcImpl) ListProductionScheduleLines(ctx context.Context, req *ListProductionScheduleLinesRequest) (*apiresource.List[apiresource.ProductionScheduleLine], *apierror.APIError) {
	pbReq := &pb.ListProductionScheduleLinesRequest{
		ProductionScheduleId: req.ProductionScheduleID,
		MachineIds:           req.MachineIDs,
		WeekIndex:            req.WeekIndex,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedules.list_lines", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListProductionScheduleLinesResponse, error) {
			return m.coreClient.ListProductionScheduleLines(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	lines := make([]apiresource.ProductionScheduleLine, len(resp.Lines))
	for i, l := range resp.Lines {
		lines[i] = scheduleLineFromProto(l)
	}
	return apiresource.NewList(lines, apiresource.PageInfo{}), nil
}

// scheduleLineFromProto is shared by the list read and every write path, so an edited line can never come back shaped differently from the same line in a list.
func scheduleLineFromProto(l *pb.ProductionScheduleLineInfo) apiresource.ProductionScheduleLine {
	return apiresource.ProductionScheduleLine{
		ID:                       l.Id,
		Object:                   constants.ObjectTypeProductionScheduleLine,
		ProductionSchedule:       entityRef(l.ProductionScheduleId, constants.ObjectTypeProductionSchedule),
		WeekIndex:                l.WeekIndex,
		WeekStartDate:            grpcutil.TimestampToTime(l.WeekStartDate),
		Machine:                  entityRef(l.MachineId, constants.ObjectTypeMachine),
		ProductionStep:           entityRefPtr(l.ProductionStepId, constants.ObjectTypeProductionStep),
		Department:               entityRefPtr(l.DepartmentId, constants.ObjectTypeDepartment),
		Item:                     entityRef(l.ItemId, constants.ObjectTypeItem),
		PlannedQuantity:          l.PlannedQuantity,
		PlannedUnit:              entityRefPtr(l.PlannedUnitId, constants.ObjectTypeUnit),
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
		Status:                   constants.ProductionScheduleLineStatus(l.StatusCode),
		Source:                   constants.ScheduleLineSource(l.SourceCode),
		Reason:                   constants.ScheduleChangeReasonPtr(l.ReasonCode),
		FreezeStatus:             constants.FreezeStatusOf(l.IsFrozen),
		ProductionRun:            entityRefPtr(l.ProductionRunId, constants.ObjectTypeProductionRun),
		CreatedAt:                grpcutil.TimestampToTime(l.CreatedAt),
		UpdatedAt:                grpcutil.TimestampToTime(l.UpdatedAt),
	}
}

func (m *productionScheduleSvcImpl) ListProductionScheduleItemPolicies(ctx context.Context, req *ListProductionScheduleItemPoliciesRequest) (*apiresource.List[apiresource.ProductionScheduleItemPolicy], *apierror.APIError) {
	pbReq := &pb.ListProductionScheduleItemPoliciesRequest{ProductionScheduleId: req.ProductionScheduleID}

	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedules.list_item_policies", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListProductionScheduleItemPoliciesResponse, error) {
			return m.coreClient.ListProductionScheduleItemPolicies(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	policies := make([]apiresource.ProductionScheduleItemPolicy, len(resp.Policies))
	for i, p := range resp.Policies {
		policies[i] = apiresource.ProductionScheduleItemPolicy{
			ID:                      p.Id,
			Object:                  constants.ObjectTypeProductionScheduleItemPolicy,
			ProductionSchedule:      entityRef(p.ProductionScheduleId, constants.ObjectTypeProductionSchedule),
			Item:                    entityRef(p.ItemId, constants.ObjectTypeItem),
			SKU:                     p.Sku,
			Unit:                    entityRefPtr(p.UnitId, constants.ObjectTypeUnit),
			UnitAbbreviation:        p.UnitAbbreviation,
			ProductionStep:          entityRefPtr(p.ProductionStepId, constants.ObjectTypeProductionStep),
			PrimaryMachine:          entityRefPtr(p.PrimaryMachineId, constants.ObjectTypeMachine),
			AnnualDemand:            p.AnnualDemand,
			WeeklyDemand:            p.WeeklyDemand,
			SecondsPerUnit:          p.SecondsPerUnit,
			UnitCost:                p.UnitCost,
			SetupCost:               p.SetupCost,
			HoldingCost:             p.HoldingCost,
			EOQUnits:                p.EoqUnits,
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
			ProjectedOnHand:         orEmptyFloats(p.ProjectedOnHand),
			WeeksOfCover:            p.WeeksOfCover,
			AnnualRunHours:          p.AnnualRunHours,
			ABCClass:                abcClassPtrFrom(p.AbcClass),
			FulfillmentPolicy:       constants.FulfillmentPolicy(p.FulfillmentPolicyCode),
			PolicySource:            constants.FulfillmentPolicySource(p.PolicySourceCode),
			FirmDemandUnits:         p.FirmDemandUnits,
			ForecastDemandUnits:     p.ForecastDemandUnits,
			Constraints:             policyConstraints(p.WasEoqCapped, p.WasCapacityStarved),
			CreatedAt:               grpcutil.TimestampToTime(p.CreatedAt),
			UpdatedAt:               grpcutil.TimestampToTime(p.UpdatedAt),
		}
	}
	return apiresource.NewList(policies, apiresource.PageInfo{}), nil
}

// scheduleFromProto maps a persisted schedule. The settings and diagnostics blobs are stored as JSON text; they are decoded here so clients get real objects rather than an escaped string they have to parse themselves.
func scheduleFromProto(info *pb.ProductionScheduleInfo) *apiresource.ProductionSchedule {
	if info == nil {
		return nil
	}
	s := &apiresource.ProductionSchedule{
		ID:                    info.Id,
		Object:                constants.ObjectTypeProductionSchedule,
		Version:               info.Version,
		Status:                constants.ProductionScheduleStatus(info.StatusCode),
		Name:                  info.Name,
		PlanningAsOf:          grpcutil.TimestampToTime(info.PlanningAsOf),
		HorizonStartDate:      grpcutil.TimestampToTime(info.HorizonStartDate),
		HorizonEndDate:        grpcutil.TimestampToTime(info.HorizonEndDate),
		HorizonWeeks:          info.HorizonWeeks,
		FrozenWeeks:           info.FrozenWeeks,
		FrozenThroughDate:     grpcutil.TimestampToTimePtr(info.FrozenThroughDate),
		DemandBasis:           constants.ScheduleDemandBasis(info.DemandBasisCode),
		GenerationSource:      constants.ScheduleGenerationSource(info.GenerationSourceCode),
		SolverVersion:         info.SolverVersion,
		SettingsSnapshot:      decodeJSONObject(info.SettingsSnapshot),
		Diagnostics:           decodeScheduleDiagnostics(info.Diagnostics),
		ErrorMessage:          info.ErrorMessage,
		FrozenLineCount:       info.FrozenLineCount,
		FrozenPlannedQuantity: info.FrozenPlannedQuantity,
		GeneratedBy:           actorRefFromIDPtr(info.GeneratedById),
		PublishedBy:           actorRefFromIDPtr(info.PublishedById),
		PublishedAt:           grpcutil.TimestampToTimePtr(info.PublishedAt),
		SupersededBy:          entityRefPtr(info.SupersededById, constants.ObjectTypeProductionSchedule),
		CreatedAt:             grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:             grpcutil.TimestampToTime(info.UpdatedAt),
	}
	return s
}

// actorRefFromIDPtr builds a minimal actor reference from an optional bare identity-actor id, nil when absent.
func actorRefFromIDPtr(id *string) *apiresource.Actor {
	if id == nil {
		return nil
	}
	return resourceloaders.ActorRefFromID(*id)
}

// decodeJSONObject turns stored JSON text into an object, falling back to an empty object rather than failing the whole read over a malformed snapshot.
func decodeJSONObject(raw string) map[string]any {
	if raw == "" {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return map[string]any{}
	}
	return out
}

// emptyScheduleDiagnostics returns a diagnostics object whose slices serialize as [] rather than null, matching the preview path.
func emptyScheduleDiagnostics() apiresource.ScheduleDiagnostics {
	return apiresource.ScheduleDiagnostics{
		EOQCappedSKUs:       []string{},
		UnschedulableSKUs:   []string{},
		CapacityStarvedSKUs: []string{},
		ItemsWithoutRunRate: []string{},
		AppliedOverrides:    apiresource.NewList([]apiresource.ScheduleAppliedOverride{}, apiresource.PageInfo{}),
		AtRiskOrders:        apiresource.NewList([]apiresource.ScheduleAtRiskOrder{}, apiresource.PageInfo{}),
		Finishing:           emptyScheduleFinishingDiagnostics(),
	}
}

// nonNilFloats and nonNilStrings keep a diagnostics collection serializing as [] rather than null, so a caller mapping over it never has to guard.
func nonNilFloats(v []float64) []float64 {
	if v == nil {
		return []float64{}
	}
	return v
}

func nonNilStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

// emptyScheduleFinishingDiagnostics keeps the second stage's lists serializing as [] rather than null, for the same reason the rest of the diagnostics do: a caller mapping over "what could not be made" should get an empty list, not a null it has to guard.
func emptyScheduleFinishingDiagnostics() apiresource.ScheduleFinishingDiagnostics {
	return apiresource.ScheduleFinishingDiagnostics{
		PlannedHoursByWeek:  []float64{},
		UtilisationByWeek:   []float64{},
		GreigeStarvedSKUs:   []string{},
		CapacityStarvedSKUs: []string{},
		ItemsWithoutRunRate: []string{},
	}
}

// Legacy solver snapshots were marshaled without json tags (PascalCase keys). Remap those aliases onto the public snake_case shape so retrieve matches preview.
var scheduleDiagnosticsKeyAliases = map[string]string{
	"EOQCappedSKUs":          "eoq_capped_skus",
	"UnschedulableSKUs":      "unschedulable_skus",
	"CapacityStarvedSKUs":    "capacity_starved_skus",
	"ItemsWithoutRunRate":    "items_without_run_rate",
	"ExcludedItemCount":      "excluded_item_count",
	"ConstraintMachineCount": "constraint_machine_count",
	"MeasuredBatchCount":     "measured_batch_count",
	"MachinesWithoutStep":    "machines_without_step",
	"ChangeoverSlopeMinutes": "changeover_slope_minutes",
	"AverageInputsAdded":     "average_inputs_added",
	"AppliedOverrides":       "applied_overrides",
	"FirmDemandUnits":        "firm_demand_units",
	"UndatedFirmOrderCount":  "undated_firm_order_count",
	"MakeToOrderItemCount":   "make_to_order_item_count",
	"AtRiskOrders":           "at_risk_orders",
}

var scheduleAppliedOverrideKeyAliases = map[string]string{
	"OverrideID":  "override_id",
	"ItemID":      "item_id",
	"MonthStart":  "month_start",
	"Before":      "before",
	"After":       "after",
	"TypeCode":    "adjustment",
	"ReasonCode":  "reason",
	"type_code":   "adjustment",
	"reason_code": "reason",
}

func remapJSONKeys(m map[string]any, aliases map[string]string) {
	for from, to := range aliases {
		v, ok := m[from]
		if !ok {
			continue
		}
		if _, exists := m[to]; !exists {
			m[to] = v
		}
		delete(m, from)
	}
}

// decodeScheduleDiagnostics turns the stored diagnostics blob into the typed API resource. Accepts both snake_case (current) and PascalCase (legacy) keys.
func decodeScheduleDiagnostics(raw string) apiresource.ScheduleDiagnostics {
	out := emptyScheduleDiagnostics()
	if raw == "" {
		return out
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil || m == nil {
		return out
	}
	remapJSONKeys(m, scheduleDiagnosticsKeyAliases)
	overrides := decodeAppliedOverrides(m["applied_overrides"])
	delete(m, "applied_overrides")
	atRisk := decodeAtRiskOrders(m["at_risk_orders"])
	delete(m, "at_risk_orders")
	normalized, err := json.Marshal(m)
	if err != nil {
		return out
	}
	if err := json.Unmarshal(normalized, &out); err != nil {
		out = emptyScheduleDiagnostics()
	}
	out.AppliedOverrides = apiresource.NewList(overrides, apiresource.PageInfo{})
	out.AtRiskOrders = apiresource.NewList(atRisk, apiresource.PageInfo{})
	if out.EOQCappedSKUs == nil {
		out.EOQCappedSKUs = []string{}
	}
	if out.UnschedulableSKUs == nil {
		out.UnschedulableSKUs = []string{}
	}
	if out.CapacityStarvedSKUs == nil {
		out.CapacityStarvedSKUs = []string{}
	}
	if out.ItemsWithoutRunRate == nil {
		out.ItemsWithoutRunRate = []string{}
	}
	// A version generated before the second stage existed has no finishing block at all, so its lists decode as nil. They read the same as an empty stage, which is what a plan with nothing to finish is.
	out.Finishing.PlannedHoursByWeek = nonNilFloats(out.Finishing.PlannedHoursByWeek)
	out.Finishing.UtilisationByWeek = nonNilFloats(out.Finishing.UtilisationByWeek)
	out.Finishing.GreigeStarvedSKUs = nonNilStrings(out.Finishing.GreigeStarvedSKUs)
	out.Finishing.CapacityStarvedSKUs = nonNilStrings(out.Finishing.CapacityStarvedSKUs)
	out.Finishing.ItemsWithoutRunRate = nonNilStrings(out.Finishing.ItemsWithoutRunRate)
	return out
}

// decodeAppliedOverrides converts stored override snapshots (flat override_id/item_id keys, PascalCase in legacy blobs) into the typed sub-object shape the API serves.
func decodeAppliedOverrides(raw any) []apiresource.ScheduleAppliedOverride {
	out := []apiresource.ScheduleAppliedOverride{}
	entries, ok := raw.([]any)
	if !ok {
		return out
	}
	for _, entry := range entries {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		remapJSONKeys(m, scheduleAppliedOverrideKeyAliases)
		normalized, err := json.Marshal(m)
		if err != nil {
			continue
		}
		var flat struct {
			OverrideID string    `json:"override_id"`
			ItemID     string    `json:"item_id"`
			MonthStart time.Time `json:"month_start"`
			Before     float64   `json:"before"`
			After      float64   `json:"after"`
			Adjustment string    `json:"adjustment"`
			Reason     string    `json:"reason"`
		}
		if err := json.Unmarshal(normalized, &flat); err != nil {
			continue
		}
		out = append(out, apiresource.ScheduleAppliedOverride{
			Override:   entityRef(flat.OverrideID, constants.ObjectTypeDemandOverride),
			Item:       entityRef(flat.ItemID, constants.ObjectTypeItem),
			MonthStart: flat.MonthStart,
			Before:     flat.Before,
			After:      flat.After,
			Adjustment: constants.DemandOverrideAdjustment(flat.Adjustment),
			Reason:     constants.DemandOverrideReasonPtr(nonEmptyPtr(flat.Reason)),
		})
	}
	return out
}

func (m *productionScheduleSvcImpl) ListScheduleDeviationTypes(ctx context.Context, req *ListScheduleDeviationTypesRequest) (*apiresource.List[apiresource.ScheduleDeviationType], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedule.list_deviation_types", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListScheduleDeviationTypesResponse, error) {
			return m.coreClient.ListScheduleDeviationTypes(ctx, &pb.ListScheduleDeviationTypesRequest{}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	types := make([]apiresource.ScheduleDeviationType, len(resp.Types))
	for i, t := range resp.Types {
		types[i] = apiresource.ScheduleDeviationType{
			ID:        t.Id,
			Object:    constants.ObjectTypeScheduleDeviationType,
			Code:      constants.ScheduleDeviationType(t.Code),
			Name:      t.Name,
			CreatedAt: grpcutil.TimestampToTime(t.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(t.UpdatedAt),
		}
	}

	return apiresource.NewList(types, apiresource.PageInfo{}), nil
}

func (m *productionScheduleSvcImpl) ListProductionScheduleDeviations(ctx context.Context, req *ListProductionScheduleDeviationsRequest) (*apiresource.List[apiresource.ProductionScheduleDeviation], *apierror.APIError) {
	pbReq := &pb.ListProductionScheduleDeviationsRequest{
		ScheduleId: req.ProductionScheduleID,
		Cursor:     req.Cursor,
		Limit:      req.Limit,
		FrozenOnly: req.Frozen,
		Query:      req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedule.list_deviations", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListProductionScheduleDeviationsResponse, error) {
			return m.coreClient.ListProductionScheduleDeviations(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	deviations := make([]apiresource.ProductionScheduleDeviation, len(resp.Deviations))
	for i, d := range resp.Deviations {
		deviations[i] = deviationFromProto(d)
	}

	return apiresource.NewList(deviations, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

// deviationFromProto decodes the line snapshots back into objects. They travel as JSON text so the line shape is modelled once, on the line itself, rather than frozen a second time in the deviation contract.
func deviationFromProto(d *pb.ProductionScheduleDeviationInfo) apiresource.ProductionScheduleDeviation {
	out := apiresource.ProductionScheduleDeviation{
		ID:                 d.Id,
		Object:             constants.ObjectTypeProductionScheduleDeviation,
		ProductionSchedule: entityRef(d.ProductionScheduleId, constants.ObjectTypeProductionSchedule),
		Line:               entityRefPtr(d.ProductionScheduleLineId, constants.ObjectTypeProductionScheduleLine),
		DeviationType:      constants.ScheduleDeviationType(d.DeviationTypeCode),
		FreezeStatus:       constants.FreezeStatusOf(d.IsFrozenWeek),
		WeekIndex:          d.WeekIndex,
		Machine:            entityRefPtr(d.MachineId, constants.ObjectTypeMachine),
		Item:               entityRefPtr(d.ItemId, constants.ObjectTypeItem),
		DeltaQuantity:      d.DeltaQuantity,
		DeltaRunHours:      d.DeltaRunHours,
		Reason:             constants.ScheduleChangeReasonPtr(d.ReasonCode),
		ReasonNote:         d.ReasonNote,
		Actor:              resourceloaders.ActorRefFromID(d.ActorId),
		CreatedAt:          grpcutil.TimestampToTime(d.CreatedAt),
	}
	out.Before = rawSnapshot(d.BeforeJson)
	out.After = rawSnapshot(d.AfterJson)
	return out
}

// orEmptyFloats keeps a weekly series serializing as [] rather than null. An item the solver never projected has no weeks, which is an empty series — not a missing field.
func orEmptyFloats(v []float64) []float64 {
	if v == nil {
		return []float64{}
	}
	return v
}

// rawSnapshot passes a stored snapshot through as JSON rather than decoding and re-encoding it, so the line shape is modelled once, on the line itself. An absent snapshot stays absent: a line the change created has no before, and one it removed has no after.
func rawSnapshot(raw *string) json.RawMessage {
	if raw == nil || *raw == "" {
		return nil
	}
	if !json.Valid([]byte(*raw)) {
		// A malformed snapshot must not take the whole deviation list down with it — the deltas and the reason are the load-bearing parts.
		return nil
	}
	return json.RawMessage(*raw)
}

func (m *productionScheduleSvcImpl) CreateProductionScheduleLine(ctx context.Context, req *CreateProductionScheduleLineRequest) (*apiresource.ProductionScheduleLine, *apierror.APIError) {
	pbReq := &pb.CreateProductionScheduleLineRequest{
		ScheduleId: req.ProductionScheduleID,
		WeekIndex:  req.WeekIndex,
		MachineId:  req.MachineID,
		ItemId:     req.ItemID,
		Quantity:   req.Quantity,
		Lots:       req.Lots.Ptr(),
		RunHours:   req.RunHours.Ptr(),
		ReasonCode: scheduleEnumPtr(req.Reason.Ptr()),
		ReasonNote: req.ReasonNote.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedule.create_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateProductionScheduleLineResponse, error) {
			return m.coreClient.CreateProductionScheduleLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	line := scheduleLineFromProto(resp.Line)
	return &line, nil
}

func (m *productionScheduleSvcImpl) UpdateProductionScheduleLine(ctx context.Context, req *UpdateProductionScheduleLineRequest) (*apiresource.ProductionScheduleLine, *apierror.APIError) {
	// Clearable enum: map Clearable[ScheduleChangeReason] to a StringPatch (clear vs set vs leave).
	var reasonPatch *pb.StringPatch
	switch {
	case req.Reason.IsClear():
		reasonPatch = &pb.StringPatch{Clear: true}
	case req.Reason.IsSet():
		v, _ := req.Reason.Value()
		s := string(v)
		reasonPatch = &pb.StringPatch{Value: &s}
	}

	pbReq := &pb.UpdateProductionScheduleLineRequest{
		ScheduleId:    req.ProductionScheduleID,
		LineId:        req.LineID,
		WeekIndex:     req.WeekIndex.Ptr(),
		MachineId:     req.MachineID.Ptr(),
		Quantity:      req.Quantity.Ptr(),
		Lots:          req.Lots.Ptr(),
		RunHours:      req.RunHours.Ptr(),
		SequenceIndex: req.SequenceIndex.Ptr(),
		StatusCode:    scheduleEnumPtr(req.Status.Ptr()),
		ReasonCode:    reasonPatch,
		ReasonNote:    req.ReasonNote.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedule.update_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateProductionScheduleLineResponse, error) {
			return m.coreClient.UpdateProductionScheduleLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	line := scheduleLineFromProto(resp.Line)
	return &line, nil
}

func (m *productionScheduleSvcImpl) DeleteProductionScheduleLine(ctx context.Context, req *DeleteProductionScheduleLineRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteProductionScheduleLineRequest{
		ScheduleId: req.ProductionScheduleID,
		LineId:     req.LineID,
		ReasonCode: scheduleEnumPtr(req.Reason),
		ReasonNote: req.ReasonNote,
	}

	_, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedule.delete_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteProductionScheduleLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *productionScheduleSvcImpl) PublishProductionSchedule(ctx context.Context, req *PublishProductionScheduleRequest) (*apiresource.ProductionSchedule, *apierror.APIError) {
	pbReq := &pb.PublishProductionScheduleRequest{Id: req.ProductionScheduleID}

	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedule.publish", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.PublishProductionScheduleResponse, error) {
			return m.coreClient.PublishProductionSchedule(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	schedule := *scheduleFromProto(resp.Schedule)
	return &schedule, nil
}

func (m *productionScheduleSvcImpl) ArchiveProductionSchedule(ctx context.Context, req *ArchiveProductionScheduleRequest) (*apiresource.ProductionSchedule, *apierror.APIError) {
	pbReq := &pb.ArchiveProductionScheduleRequest{Id: req.ProductionScheduleID}

	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedule.archive", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ArchiveProductionScheduleResponse, error) {
			return m.coreClient.ArchiveProductionSchedule(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	schedule := *scheduleFromProto(resp.Schedule)
	return &schedule, nil
}

func (m *productionScheduleSvcImpl) DeleteProductionSchedule(ctx context.Context, req *DeleteProductionScheduleRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteProductionScheduleRequest{Id: req.ProductionScheduleID}

	_, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedule.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteProductionSchedule(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

// nonEmptyPtr turns an unset proto string into nil, so "no reason recorded" stays distinguishable from a reason that happens to be blank.
func nonEmptyPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

// scheduleEnumPtr narrows a typed-enum pointer to the plain string pointer the proto layer uses. Storage keeps the untyped vocabulary; typing lives at the API boundary.
func scheduleEnumPtr[T ~string](v *T) *string {
	if v == nil {
		return nil
	}
	s := string(*v)
	return &s
}

// abcClassPtr returns nil for an unclassified item rather than an empty class, so "not ranked" stays distinguishable from a real rank.
func abcClassPtr(value string) *constants.ABCClass {
	if value == "" {
		return nil
	}
	class := constants.ABCClass(value)
	return &class
}

func abcClassPtrFrom(value *string) *constants.ABCClass {
	if value == nil {
		return nil
	}
	return abcClassPtr(*value)
}

// policyConstraints turns the stored diagnostic flags into a named list. A new limit then adds a value rather than another boolean on every policy row.
func policyConstraints(eoqCapped, capacityStarved bool) []constants.SchedulePolicyConstraint {
	out := []constants.SchedulePolicyConstraint{}
	if eoqCapped {
		out = append(out, constants.SchedulePolicyConstraintEOQCapped)
	}
	if capacityStarved {
		out = append(out, constants.SchedulePolicyConstraintCapacityStarved)
	}
	return out
}

// enumStrings narrows typed enum filters to the plain strings the proto layer carries.
func enumStrings[T ~string](values []T) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

func (m *productionScheduleSvcImpl) ListProductionScheduleDerivedLines(ctx context.Context, req *ListProductionScheduleDerivedLinesRequest) (*apiresource.List[apiresource.ProductionScheduleDerivedLine], *apierror.APIError) {
	pbReq := &pb.ListProductionScheduleDerivedLinesRequest{
		ScheduleId:    req.ProductionScheduleID,
		DepartmentIds: req.DepartmentIDs,
		WeekIndex:     req.WeekIndex,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedule.list_derived_lines", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListProductionScheduleDerivedLinesResponse, error) {
			return m.coreClient.ListProductionScheduleDerivedLines(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	lines := make([]apiresource.ProductionScheduleDerivedLine, len(resp.Lines))
	for i, l := range resp.Lines {
		lines[i] = apiresource.ProductionScheduleDerivedLine{
			ID:                 l.Id,
			Object:             constants.ObjectTypeProductionScheduleDerivedLine,
			ProductionSchedule: entityRef(l.ProductionScheduleId, constants.ObjectTypeProductionSchedule),
			SourceLine:         entityRef(l.SourceLineId, constants.ObjectTypeProductionScheduleLine),
			ProductionStep:     entityRef(l.ProductionStepId, constants.ObjectTypeProductionStep),
			Department:         entityRefPtr(l.DepartmentId, constants.ObjectTypeDepartment),
			Item:               entityRef(l.ItemId, constants.ObjectTypeItem),
			WeekIndex:          l.WeekIndex,
			WeekStartDate:      grpcutil.TimestampToTime(l.WeekStartDate),
			Quantity:           l.Quantity,
			PlannedUnit:        entityRefPtr(l.PlannedUnitId, constants.ObjectTypeUnit),
			ExplosionDepth:     l.ExplosionDepth,
			OffsetWeeks:        l.OffsetWeeks,
			Status:             constants.ProductionScheduleLineStatus(l.StatusCode),
			CreatedAt:          grpcutil.TimestampToTime(l.CreatedAt),
			UpdatedAt:          grpcutil.TimestampToTime(l.UpdatedAt),
		}
	}

	return apiresource.NewList(lines, apiresource.PageInfo{}), nil
}

// regenerateProtoRequest is built once for both the preview and the apply so the two can never send a different question to the solver.
func regenerateProtoRequest(
	scheduleID string,
	mergeMode *constants.ScheduleMergeMode,
	planningAsOf *time.Time,
	horizonWeeks *int32,
	demandBasis *constants.ScheduleDemandBasis,
) *pb.RegenerateProductionScheduleRequest {
	pbReq := &pb.RegenerateProductionScheduleRequest{
		Id:           scheduleID,
		MergeMode:    scheduleEnumPtr(mergeMode),
		HorizonWeeks: horizonWeeks,
		DemandBasis:  scheduleEnumPtr(demandBasis),
	}
	if planningAsOf != nil {
		pbReq.PlanningAsOf = timestamppb.New(*planningAsOf)
	}
	return pbReq
}

func (m *productionScheduleSvcImpl) PreviewRegenerateProductionSchedule(ctx context.Context, req *PreviewRegenerateProductionScheduleRequest) (*apiresource.ProductionScheduleRegeneratePreview, *apierror.APIError) {
	pbReq := regenerateProtoRequest(req.ProductionScheduleID, nil, req.PlanningAsOf.Ptr(), req.HorizonWeeks.Ptr(), req.DemandBasis.Ptr())

	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedule.preview_regenerate", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.PreviewRegenerateProductionScheduleResponse, error) {
			return m.coreClient.PreviewRegenerateProductionSchedule(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	// Initialized rather than left nil so the list serializes as [] and a client never has to tell "nothing would change" apart from "the field is missing".
	lines := make([]apiresource.ScheduleDiffLine, len(resp.Lines))
	for i, line := range resp.Lines {
		lines[i] = apiresource.ScheduleDiffLine{
			Change:           constants.ScheduleDiffChange(line.ChangeCode),
			Item:             entityRef(line.ItemId, constants.ObjectTypeItem),
			SKU:              line.Sku,
			Machine:          entityRef(line.MachineId, constants.ObjectTypeMachine),
			WeekIndex:        line.WeekIndex,
			CurrentQuantity:  line.CurrentQuantity,
			ProposedQuantity: line.ProposedQuantity,
			CurrentIsManual:  line.CurrentIsManual,
		}
	}

	return &apiresource.ProductionScheduleRegeneratePreview{
		Object:               constants.ObjectTypeProductionScheduleRegeneratePreview,
		ProductionSchedule:   entityRef(resp.ScheduleId, constants.ObjectTypeProductionSchedule),
		SolverVersion:        resp.SolverVersion,
		PlanningAsOf:         grpcutil.TimestampToTime(resp.PlanningAsOf),
		Lines:                apiresource.NewList(lines, apiresource.PageInfo{}),
		AddedCount:           resp.AddedCount,
		RemovedCount:         resp.RemovedCount,
		ChangedCount:         resp.ChangedCount,
		ManualLineCount:      resp.ManualLineCount,
		DiscardedManualCount: resp.DiscardedManualCount,
	}, nil
}

func (m *productionScheduleSvcImpl) RegenerateProductionSchedule(ctx context.Context, req *RegenerateProductionScheduleRequest) (*apiresource.ProductionSchedule, *apierror.APIError) {
	pbReq := regenerateProtoRequest(req.ProductionScheduleID, req.MergeMode.Ptr(), req.PlanningAsOf.Ptr(), req.HorizonWeeks.Ptr(), req.DemandBasis.Ptr())

	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedule.regenerate", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.RegenerateProductionScheduleResponse, error) {
			return m.coreClient.RegenerateProductionSchedule(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return scheduleFromProto(resp.Schedule), nil
}

func (m *productionScheduleSvcImpl) ListProductionScheduleFinishedPolicies(ctx context.Context, req *ListProductionScheduleFinishedPoliciesRequest) (*apiresource.List[apiresource.ProductionScheduleFinishedPolicy], *apierror.APIError) {
	pbReq := &pb.ListProductionScheduleFinishedPoliciesRequest{ProductionScheduleId: req.ProductionScheduleID}

	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedules.list_finished_policies", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListProductionScheduleFinishedPoliciesResponse, error) {
			return m.coreClient.ListProductionScheduleFinishedPolicies(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	policies := make([]apiresource.ProductionScheduleFinishedPolicy, len(resp.Policies))
	for i, p := range resp.Policies {
		policies[i] = apiresource.ProductionScheduleFinishedPolicy{
			ID:                 p.Id,
			Object:             constants.ObjectTypeProductionScheduleFinishedPolicy,
			ProductionSchedule: entityRef(p.ProductionScheduleId, constants.ObjectTypeProductionSchedule),
			Item:               entityRef(p.ItemId, constants.ObjectTypeItem),
			SKU:                p.Sku,
			GreigeItem:         entityRef(p.GreigeItemId, constants.ObjectTypeItem),
			GreigeSKU:          p.GreigeSku,
			ProductLine:        entityRefPtr(p.ProductLineId, constants.ObjectTypeProductLine),
			AnnualDemand:       p.AnnualDemand,
			WeeklyDemand:       p.WeeklyDemand,
			SigmaWeekly:        p.SigmaWeekly,
			SafetyStock:        p.SafetyStock,
			ReorderPoint:       p.ReorderPoint,
			OnHand:             p.OnHand,
			WeeksOfCover:       p.WeeksOfCover,
			CreatedAt:          grpcutil.TimestampToTime(p.CreatedAt),
			UpdatedAt:          grpcutil.TimestampToTime(p.UpdatedAt),
		}
	}

	return &apiresource.List[apiresource.ProductionScheduleFinishedPolicy]{
		Object: constants.ObjectTypeList,
		Data:   policies,
	}, nil
}

func (m *productionScheduleSvcImpl) ListProductionScheduleFinishingLines(ctx context.Context, req *ListProductionScheduleFinishingLinesRequest) (*apiresource.List[apiresource.ProductionScheduleFinishingLine], *apierror.APIError) {
	pbReq := &pb.ListProductionScheduleFinishingLinesRequest{
		ProductionScheduleId: req.ProductionScheduleID,
		WeekIndex:            req.WeekIndex,
		ItemId:               req.ItemID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedules.list_finishing_lines", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListProductionScheduleFinishingLinesResponse, error) {
			return m.coreClient.ListProductionScheduleFinishingLines(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	lines := make([]apiresource.ProductionScheduleFinishingLine, len(resp.Lines))
	for i, l := range resp.Lines {
		lines[i] = apiresource.ProductionScheduleFinishingLine{
			ID:                    l.Id,
			Object:                constants.ObjectTypeProductionScheduleFinishingLine,
			ProductionSchedule:    entityRef(l.ProductionScheduleId, constants.ObjectTypeProductionSchedule),
			WeekIndex:             l.WeekIndex,
			WeekStartDate:         grpcutil.TimestampToTime(l.WeekStartDate),
			Item:                  entityRef(l.ItemId, constants.ObjectTypeItem),
			SKU:                   l.Sku,
			GreigeItem:            entityRef(l.GreigeItemId, constants.ObjectTypeItem),
			GreigeSKU:             l.GreigeSku,
			Department:            entityRefPtr(l.DepartmentId, constants.ObjectTypeDepartment),
			ProductionStep:        entityRefPtr(l.ProductionStepId, constants.ObjectTypeProductionStep),
			PlannedQuantity:       l.PlannedQuantity,
			Unit:                  l.PlannedUnitAbbreviation,
			PlannedLots:           l.PlannedLots,
			PlannedLotUnits:       l.PlannedLotUnits,
			PlannedRunHours:       l.PlannedRunHours,
			GreigeConsumed:        l.GreigeConsumed,
			FirmUnits:             l.FirmUnits,
			ProjectedOnHandBefore: l.ProjectedOnHandBefore,
			ProjectedOnHandAfter:  l.ProjectedOnHandAfter,
			Status:                constants.ProductionScheduleLineStatus(l.StatusCode),
			Source:                constants.ScheduleLineSource(l.SourceCode),
			IsFrozen:              l.IsFrozen,
			CreatedAt:             grpcutil.TimestampToTime(l.CreatedAt),
			UpdatedAt:             grpcutil.TimestampToTime(l.UpdatedAt),
		}
	}

	return &apiresource.List[apiresource.ProductionScheduleFinishingLine]{
		Object: constants.ObjectTypeList,
		Data:   lines,
	}, nil
}

// boolOrFalse reads an optional flag, where omitting it means the default behaviour rather than a third state.
func boolOrFalse(f field.Optional[bool]) bool {
	if v := f.Ptr(); v != nil {
		return *v
	}
	return false
}

func releasedLineFromProto(info *pb.ReleasedScheduleLineInfo) apiresource.ReleasedScheduleLine {
	batches := make([]apiresource.ReleaseScheduleBatch, 0, len(info.Batches))
	for _, b := range info.Batches {
		batches = append(batches, apiresource.ReleaseScheduleBatch{
			Item:               entityRef(b.ItemId, constants.ObjectTypeItem),
			SKU:                b.Sku,
			Quantity:           b.Quantity,
			Batch:              entityRefPtr(b.BatchId, constants.ObjectTypeBatch),
			CarriedForwardFrom: b.CarriedForwardFrom,
		})
	}

	return apiresource.ReleasedScheduleLine{
		Line:                   entityRef(info.ProductionScheduleLineId, constants.ObjectTypeProductionScheduleLine),
		Item:                   entityRef(info.ItemId, constants.ObjectTypeItem),
		SKU:                    info.Sku,
		Machine:                apiresource.NewEntity(info.MachineId, constants.ObjectTypeMachine, info.MachineName, nil),
		PlannedQuantity:        info.PlannedQuantity,
		LotUnits:               info.LotUnits,
		Unit:                   info.Unit,
		BatchCount:             info.BatchCount,
		CarriedForwardQuantity: info.CarriedForwardQuantity,
		Batches:                apiresource.NewList(batches, apiresource.PageInfo{}),
	}
}

func releasedLinesFromProto(infos []*pb.ReleasedScheduleLineInfo) []apiresource.ReleasedScheduleLine {
	// Empty rather than nil: a caller mapping over these should get an empty list.
	lines := make([]apiresource.ReleasedScheduleLine, 0, len(infos))
	for _, info := range infos {
		lines = append(lines, releasedLineFromProto(info))
	}
	return lines
}

func (m *productionScheduleSvcImpl) ReleaseProductionScheduleWeek(ctx context.Context, req *ReleaseProductionScheduleWeekRequest) (*apiresource.ReleaseScheduleWeekResult, *apierror.APIError) {
	pbReq := &pb.ReleaseProductionScheduleWeekRequest{
		Id:                req.ProductionScheduleID,
		WeekIndex:         req.WeekIndex,
		ResponsibleUserId: req.ResponsibleUserID,
		SkipCarryForward:  boolOrFalse(req.SkipCarryForward),
	}
	pbReq.ScanningStationId = req.ScanningStationID.Ptr()

	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedule.release_week", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ReleaseProductionScheduleWeekResponse, error) {
			return m.coreClient.ReleaseProductionScheduleWeek(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := &apiresource.ReleaseScheduleWeekResult{
		Object:                   constants.ObjectTypeProductionScheduleWeekRelease,
		WeekIndex:                resp.WeekIndex,
		WeekStartDate:            grpcutil.TimestampToTime(resp.WeekStartDate),
		ReleasedLineCount:        resp.ReleasedLineCount,
		BatchCount:               resp.BatchCount,
		CarriedForwardBatchCount: resp.CarriedForwardBatchCount,
		TotalQuantity:            resp.TotalQuantity,
		Lines:                    apiresource.NewList(releasedLinesFromProto(resp.Lines), apiresource.PageInfo{}),
	}
	if resp.ProductionRun != nil {
		run := productionrunep.ProductionRunFromProto(resp.ProductionRun)
		result.ProductionRun = &run
	}

	return result, nil
}

func (m *productionScheduleSvcImpl) PreviewReleaseProductionScheduleWeek(ctx context.Context, req *PreviewReleaseProductionScheduleWeekRequest) (*apiresource.ReleaseScheduleWeekPreview, *apierror.APIError) {
	pbReq := &pb.PreviewReleaseProductionScheduleWeekRequest{
		Id:               req.ProductionScheduleID,
		WeekIndex:        req.WeekIndex,
		SkipCarryForward: req.SkipCarryForward,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedule.preview_release_week", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ReleaseProductionScheduleWeekPreviewResponse, error) {
			return m.coreClient.PreviewReleaseProductionScheduleWeek(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.ReleaseScheduleWeekPreview{
		Object:                   constants.ObjectTypeProductionScheduleWeekReleasePreview,
		WeekIndex:                resp.WeekIndex,
		WeekStartDate:            grpcutil.TimestampToTime(resp.WeekStartDate),
		LineCount:                resp.LineCount,
		BatchCount:               resp.BatchCount,
		CarriedForwardBatchCount: resp.CarriedForwardBatchCount,
		TotalQuantity:            resp.TotalQuantity,
		Lines:                    apiresource.NewList(releasedLinesFromProto(resp.Lines), apiresource.PageInfo{}),
		IsReleasable:             resp.IsReleasable,
		BlockedReason:            resp.BlockedReason,
		ExistingProductionRun:    entityRefPtr(resp.ExistingProductionRunId, constants.ObjectTypeProductionRun),
	}, nil
}

// decodeAtRiskOrders converts the stored at-risk snapshot (flat sales_order_id/item_id keys) into the typed sub-object shape the API serves.
//
// A version solved before the order book existed carries no at_risk_orders key, which decodes to an empty list rather than null: a plan that met every commitment and a plan from before commitments were tracked both have nothing to act on, and should read the same.
func decodeAtRiskOrders(raw any) []apiresource.ScheduleAtRiskOrder {
	out := []apiresource.ScheduleAtRiskOrder{}
	entries, ok := raw.([]any)
	if !ok {
		return out
	}
	for _, entry := range entries {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		normalized, err := json.Marshal(m)
		if err != nil {
			continue
		}
		var flat struct {
			SalesOrderID     string  `json:"sales_order_id"`
			SalesOrderNumber string  `json:"sales_order_number"`
			ItemID           string  `json:"item_id"`
			SKU              string  `json:"sku"`
			Units            float64 `json:"units"`
			DueWeek          int32   `json:"due_week"`
			Reason           string  `json:"reason"`
		}
		if err := json.Unmarshal(normalized, &flat); err != nil {
			continue
		}
		order := apiresource.ScheduleAtRiskOrder{
			Object:  constants.ObjectTypeScheduleAtRiskOrder,
			SKU:     flat.SKU,
			Units:   flat.Units,
			DueWeek: flat.DueWeek,
			Reason:  constants.ScheduleAtRiskReason(flat.Reason),
		}
		order.SalesOrder = entityRef(flat.SalesOrderID, constants.ObjectTypeSalesOrder)
		if flat.SalesOrderNumber != "" && order.SalesOrder != nil {
			order.SalesOrder = apiresource.NewEntity(flat.SalesOrderID, constants.ObjectTypeSalesOrder, nil, &flat.SalesOrderNumber)
		}
		if flat.ItemID != "" {
			label := flat.SKU
			order.Item = apiresource.NewEntity(flat.ItemID, constants.ObjectTypeItem, nil, &label)
		}
		out = append(out, order)
	}
	return out
}

func (m *productionScheduleSvcImpl) ListAtRiskOrders(ctx context.Context, req *ListAtRiskOrdersRequest) (*apiresource.List[apiresource.ScheduleOrderCoverage], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedule.list_at_risk_orders", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListProductionScheduleAtRiskOrdersResponse, error) {
			return m.coreClient.ListProductionScheduleAtRiskOrders(ctx, &pb.ListProductionScheduleAtRiskOrdersRequest{ProductionScheduleId: req.ScheduleID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	items := make([]apiresource.ScheduleOrderCoverage, 0, len(resp.Orders))
	for _, o := range resp.Orders {
		covering := make([]apiresource.ScheduleOrderCoverageLine, 0, len(o.CoveringLines))
		for _, l := range o.CoveringLines {
			covering = append(covering, apiresource.ScheduleOrderCoverageLine{
				Object:                 constants.ObjectTypeScheduleOrderCoverageLine,
				ProductionScheduleLine: entityRef(l.ProductionScheduleLineId, constants.ObjectTypeProductionScheduleLine),
				WeekIndex:              l.WeekIndex,
				Machine:                entityRef(l.MachineId, constants.ObjectTypeMachine),
				AllocatedQuantity:      l.AllocatedQuantity,
			})
		}
		row := apiresource.ScheduleOrderCoverage{
			Object:        constants.ObjectTypeScheduleOrderCoverage,
			SalesOrder:    apiresource.NewEntity(o.SalesOrderId, constants.ObjectTypeSalesOrder, nil, &o.SalesOrderNumber),
			Item:          apiresource.NewEntity(o.ItemId, constants.ObjectTypeItem, nil, &o.Sku),
			SKU:           o.Sku,
			UnitsAtRisk:   o.UnitsAtRisk,
			DueWeek:       o.DueWeek,
			Reason:        constants.ScheduleAtRiskReason(o.ReasonCode),
			CoveringLines: apiresource.NewList(covering, apiresource.PageInfo{}),
		}
		if o.ShipByDate != nil {
			t := grpcutil.TimestampToTime(o.ShipByDate)
			row.ShipByDate = &t
		}
		items = append(items, row)
	}
	return apiresource.NewList(items, apiresource.PageInfo{}), nil
}

func (m *productionScheduleSvcImpl) QuotePromiseDate(ctx context.Context, req *QuotePromiseDateRequest) (*apiresource.PromiseDateQuote, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, productionScheduleEpSvcTracer, "service.production_schedule.quote_promise_date", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.QuotePromiseDateResponse, error) {
			return m.coreClient.QuotePromiseDate(ctx, &pb.QuotePromiseDateRequest{ItemId: req.ItemID, Quantity: req.Quantity}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	out := &apiresource.PromiseDateQuote{
		Object:                    constants.ObjectTypePromiseDateQuote,
		Item:                      entityRef(resp.ItemId, constants.ObjectTypeItem),
		Quantity:                  resp.Quantity,
		IsPromisable:              resp.IsPromisable,
		EarliestWeekIndex:         resp.EarliestWeekIndex,
		ProductionSchedule:        entityRef(resp.ProductionScheduleId, constants.ObjectTypeProductionSchedule),
		ProductionScheduleVersion: resp.ProductionScheduleVersion,
	}
	if resp.EarliestShipDate != nil {
		t := grpcutil.TimestampToTime(resp.EarliestShipDate)
		out.EarliestShipDate = &t
	}
	return out, nil
}
