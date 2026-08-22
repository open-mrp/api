package analyticsep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/safeconv"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AnalyticsSvc interface {
	AnalyzeSales(ctx context.Context, req *AnalyzeSalesRequest) (*apiresource.AnalyzeSalesResponse, *apierror.APIError)
	AnalyzeOpenBatches(ctx context.Context, req *AnalyzeOpenBatchesRequest) (*apiresource.AnalyzeOpenBatchesResponse, *apierror.APIError)
	AnalyzeProductionCosts(ctx context.Context, req *AnalyzeProductionCostsRequest) (*apiresource.AnalyzeProductionCostsResponse, *apierror.APIError)
	AnalyzeDeliveries(ctx context.Context, req *AnalyzeDeliveriesRequest) (*apiresource.AnalyzeDeliveriesResponse, *apierror.APIError)
	AnalyzeManufacturing(ctx context.Context, req *AnalyzeManufacturingRequest) (*apiresource.AnalyzeManufacturingResponse, *apierror.APIError)
	AnalyzeManufacturingBatch(ctx context.Context, req *AnalyzeManufacturingBatchRequest) (*apiresource.AnalyzeManufacturingBatchResponse, *apierror.APIError)
	AnalyzeOrders(ctx context.Context, req *AnalyzeOrdersRequest) (*apiresource.AnalyzeOrdersResponse, *apierror.APIError)
	AnalyzeQuarterlyOrders(ctx context.Context, req *AnalyzeQuarterlyOrdersRequest) (*apiresource.AnalyzeQuarterlyOrdersResponse, *apierror.APIError)
	AnalyzeMaterials(ctx context.Context, req *AnalyzeMaterialsRequest) (*apiresource.AnalyzeMaterialsResponse, *apierror.APIError)
	AnalyzeInventoryReceipts(ctx context.Context, req *AnalyzeInventoryReceiptsRequest) (*apiresource.AnalyzeInventoryReceiptsResponse, *apierror.APIError)
	AnalyzeNewCustomers(ctx context.Context, req *AnalyzeNewCustomersRequest) (*apiresource.AnalyzeNewCustomersResponse, *apierror.APIError)
	AnalyzeDemandForecast(ctx context.Context, req *AnalyzeDemandForecastRequest) (*apiresource.AnalyzeDemandForecastResponse, *apierror.APIError)
	AnalyzeOee(ctx context.Context, req *AnalyzeOeeRequest) (*apiresource.AnalyzeOeeResponse, *apierror.APIError)
	AnalyzeOeeTrend(ctx context.Context, req *AnalyzeOeeTrendRequest) (*apiresource.AnalyzeOeeTrendResponse, *apierror.APIError)
	AnalyzeScheduleAttainment(ctx context.Context, req *AnalyzeScheduleAttainmentRequest) (*apiresource.AnalyzeScheduleAttainmentResponse, *apierror.APIError)
	AnalyzeDeliveryPerformance(ctx context.Context, req *AnalyzeDeliveryPerformanceRequest) (*apiresource.AnalyzeDeliveryPerformanceResponse, *apierror.APIError)
	AnalyzeWeeksOfSales(ctx context.Context, req *AnalyzeWeeksOfSalesRequest) (*apiresource.AnalyzeWeeksOfSalesResponse, *apierror.APIError)
	AnalyzeCustomerPricing(ctx context.Context, req *AnalyzeCustomerPricingRequest) (*apiresource.AnalyzeCustomerPricingResponse, *apierror.APIError)
	AnalyzeRealizedMargins(ctx context.Context, req *AnalyzeRealizedMarginsRequest) (*apiresource.AnalyzeRealizedMarginsResponse, *apierror.APIError)
}

type AnalyticsSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type analyticsSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var analyticsSvcTracer = tracing.GetTracer("api-gateway.endpoints.analytics.service")

func (c *AnalyticsSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("analytics endpoint service: core client is required")
	}
	return nil
}

func NewAnalyticsSvc(config *AnalyticsSvcConfig) AnalyticsSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &analyticsSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *analyticsSvcImpl) AnalyzeSales(ctx context.Context, req *AnalyzeSalesRequest) (*apiresource.AnalyzeSalesResponse, *apierror.APIError) {
	pbReq := &pb.AnalyzeSalesRequest{
		StartDate:        timestamppb.New(req.StartDate),
		EndDate:          timestamppb.New(req.EndDate),
		ProductLineIds:   req.ProductLineIDs,
		CustomerIds:      req.CustomerIDs,
		SalesRepIds:      req.SalesRepIDs,
		CustomerGroupIds: req.CustomerGroupIDs,
	}
	if req.Query != nil {
		pbReq.Query = req.Query
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_sales", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeSalesResponse, error) {
			return m.coreClient.AnalyzeSales(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AnalyzeSalesPresenter(resp), nil
}

func (m *analyticsSvcImpl) AnalyzeOpenBatches(ctx context.Context, req *AnalyzeOpenBatchesRequest) (*apiresource.AnalyzeOpenBatchesResponse, *apierror.APIError) {
	pbReq := &pb.AnalyzeOpenBatchesRequest{
		ItemIds:        req.ItemIDs,
		ProductLineIds: req.ProductLineIDs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_open_batches", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeOpenBatchesResponse, error) {
			return m.coreClient.AnalyzeOpenBatches(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AnalyzeOpenBatchesPresenter(resp), nil
}

func (m *analyticsSvcImpl) AnalyzeProductionCosts(ctx context.Context, req *AnalyzeProductionCostsRequest) (*apiresource.AnalyzeProductionCostsResponse, *apierror.APIError) {
	pbReq := &pb.AnalyzeProductionCostsRequest{
		ItemIds:        req.ItemIDs,
		ProductLineIds: req.ProductLineIDs,
		DepartmentIds:  req.DepartmentIDs,
		CategoryIds:    req.CategoryIDs,
	}
	if req.StartDate != nil {
		pbReq.StartDate = timestamppb.New(*req.StartDate)
	}
	if req.EndDate != nil {
		pbReq.EndDate = timestamppb.New(*req.EndDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_production_costs", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeProductionCostsResponse, error) {
			return m.coreClient.AnalyzeProductionCosts(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AnalyzeProductionCostsPresenter(resp), nil
}

func (m *analyticsSvcImpl) AnalyzeDeliveries(ctx context.Context, req *AnalyzeDeliveriesRequest) (*apiresource.AnalyzeDeliveriesResponse, *apierror.APIError) {
	pbReq := &pb.AnalyzeDeliveriesRequest{
		StartDate:        timestamppb.New(req.StartDate),
		EndDate:          timestamppb.New(req.EndDate),
		ProductLineIds:   req.ProductLineIDs,
		CustomerIds:      req.CustomerIDs,
		CustomerGroupIds: req.CustomerGroupIDs,
		SalesRepIds:      req.SalesRepIDs,
	}
	if req.TargetDeliveryTimeDays != nil {
		v := safeconv.Int64ToInt32(*req.TargetDeliveryTimeDays)
		pbReq.TargetDeliveryTimeDays = &v
	}
	if req.OverridePromisedDates != nil {
		pbReq.OverridePromisedDates = req.OverridePromisedDates
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_deliveries", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeDeliveriesResponse, error) {
			return m.coreClient.AnalyzeDeliveries(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AnalyzeDeliveriesPresenter(resp), nil
}

func (m *analyticsSvcImpl) AnalyzeManufacturing(ctx context.Context, req *AnalyzeManufacturingRequest) (*apiresource.AnalyzeManufacturingResponse, *apierror.APIError) {
	pbReq := &pb.AnalyzeManufacturingRequest{
		StartDate: timestamppb.New(req.StartDate),
		EndDate:   timestamppb.New(req.EndDate),
		Type:      req.Type,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_manufacturing", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeManufacturingResponse, error) {
			return m.coreClient.AnalyzeManufacturing(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AnalyzeManufacturingPresenter(resp), nil
}

func (m *analyticsSvcImpl) AnalyzeManufacturingBatch(ctx context.Context, req *AnalyzeManufacturingBatchRequest) (*apiresource.AnalyzeManufacturingBatchResponse, *apierror.APIError) {
	pbReq := &pb.AnalyzeManufacturingBatchRequest{
		StartDate:           timestamppb.New(req.StartDate),
		EndDate:             timestamppb.New(req.EndDate),
		ComparisonStartDate: timestamppb.New(req.ComparisonStartDate),
		ComparisonEndDate:   timestamppb.New(req.ComparisonEndDate),
		CustomerIds:         req.CustomerIDs,
		ProductLineIds:      req.ProductLineIDs,
		CustomerGroupIds:    req.CustomerGroupIDs,
		ItemIds:             req.ItemIDs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_manufacturing_batch", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeManufacturingBatchResponse, error) {
			return m.coreClient.AnalyzeManufacturingBatch(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AnalyzeManufacturingBatchPresenter(resp), nil
}

func (m *analyticsSvcImpl) AnalyzeOrders(ctx context.Context, req *AnalyzeOrdersRequest) (*apiresource.AnalyzeOrdersResponse, *apierror.APIError) {
	// Determine if the caller is a sales rep from identity, matching dashboard behavior.
	var isSalesRep bool
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok && identity != nil && identity.Actor != nil && identity.Actor.RoleType != nil {
		isSalesRep = *identity.Actor.RoleType == string(constants.RoleTypeSalesRep)
	}

	pbReq := &pb.AnalyzeOrdersRequest{
		ProductLineIds:   req.ProductLineIDs,
		CustomerIds:      req.CustomerIDs,
		SalesRepIds:      req.SalesRepIDs,
		CustomerGroupIds: req.CustomerGroupIDs,
		IsSalesRep:       isSalesRep,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_orders", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeOrdersResponse, error) {
			return m.coreClient.AnalyzeOrders(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AnalyzeOrdersPresenter(resp), nil
}

func (m *analyticsSvcImpl) AnalyzeQuarterlyOrders(ctx context.Context, req *AnalyzeQuarterlyOrdersRequest) (*apiresource.AnalyzeQuarterlyOrdersResponse, *apierror.APIError) {
	pbReq := &pb.AnalyzeQuarterlyOrdersRequest{
		SalesRepIds:      req.SalesRepIDs,
		ItemIds:          req.ItemIDs,
		ProductLineIds:   req.ProductLineIDs,
		CustomerIds:      req.CustomerIDs,
		CustomerGroupIds: req.CustomerGroupIDs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_quarterly_orders", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeQuarterlyOrdersResponse, error) {
			return m.coreClient.AnalyzeQuarterlyOrders(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AnalyzeQuarterlyOrdersPresenter(resp), nil
}

func (m *analyticsSvcImpl) AnalyzeMaterials(ctx context.Context, req *AnalyzeMaterialsRequest) (*apiresource.AnalyzeMaterialsResponse, *apierror.APIError) {
	pbReq := &pb.AnalyzeMaterialsRequest{
		SalesOrderIds: req.SalesOrderIDs,
		SupplierIds:   req.SupplierIDs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_materials", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeMaterialsResponse, error) {
			return m.coreClient.AnalyzeMaterials(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AnalyzeMaterialsPresenter(resp), nil
}

func (m *analyticsSvcImpl) AnalyzeInventoryReceipts(ctx context.Context, req *AnalyzeInventoryReceiptsRequest) (*apiresource.AnalyzeInventoryReceiptsResponse, *apierror.APIError) {
	pbReq := &pb.AnalyzeInventoryReceiptsRequest{
		ItemIds:     req.ItemIDs,
		LocationIds: req.LocationIDs,
		LotIds:      req.LotIDs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_inventory_receipts", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeInventoryReceiptsResponse, error) {
			return m.coreClient.AnalyzeInventoryReceipts(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AnalyzeInventoryReceiptsPresenter(resp), nil
}

func (m *analyticsSvcImpl) AnalyzeNewCustomers(ctx context.Context, req *AnalyzeNewCustomersRequest) (*apiresource.AnalyzeNewCustomersResponse, *apierror.APIError) {
	pbReq := &pb.AnalyzeNewCustomersRequest{
		StartDate:        timestamppb.New(req.StartDate),
		EndDate:          timestamppb.New(req.EndDate),
		CustomerGroupIds: req.CustomerGroupIDs,
		SalesRepIds:      req.SalesRepIDs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_new_customers", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeNewCustomersResponse, error) {
			return m.coreClient.AnalyzeNewCustomers(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AnalyzeNewCustomersPresenter(resp), nil
}

func (m *analyticsSvcImpl) AnalyzeDemandForecast(ctx context.Context, req *AnalyzeDemandForecastRequest) (*apiresource.AnalyzeDemandForecastResponse, *apierror.APIError) {
	pbReq := &pb.AnalyzeDemandForecastRequest{
		ProductLineIds: req.ProductLineIDs,
		ItemIds:        req.ItemIDs,
	}
	if req.HistoryMonths != nil {
		v := safeconv.Int64ToInt32(*req.HistoryMonths)
		pbReq.HistoryMonths = &v
	}
	if req.ForecastMonths != nil {
		v := safeconv.Int64ToInt32(*req.ForecastMonths)
		pbReq.ForecastMonths = &v
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_demand_forecast", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeDemandForecastResponse, error) {
			return m.coreClient.AnalyzeDemandForecast(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AnalyzeDemandForecastPresenter(resp), nil
}

func (m *analyticsSvcImpl) AnalyzeOee(ctx context.Context, req *AnalyzeOeeRequest) (*apiresource.AnalyzeOeeResponse, *apierror.APIError) {
	plannedTime := make([]*pb.OeeDepartmentPlannedTimeProto, len(req.PlannedTime))
	for i, pt := range req.PlannedTime {
		plannedTime[i] = &pb.OeeDepartmentPlannedTimeProto{
			DepartmentId: pt.DepartmentID,
			PlannedHours: pt.PlannedHours,
		}
	}

	pbReq := &pb.AnalyzeOeeRequest{
		StartDate:     timestamppb.New(req.StartDate),
		EndDate:       timestamppb.New(req.EndDate),
		DepartmentIds: req.DepartmentIDs,
		PlannedTime:   plannedTime,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_oee", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeOeeResponse, error) {
			return m.coreClient.AnalyzeOee(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AnalyzeOeePresenter(resp), nil
}

func (m *analyticsSvcImpl) AnalyzeOeeTrend(ctx context.Context, req *AnalyzeOeeTrendRequest) (*apiresource.AnalyzeOeeTrendResponse, *apierror.APIError) {
	pbReq := &pb.AnalyzeOeeTrendRequest{
		StartDate:     timestamppb.New(req.StartDate),
		EndDate:       timestamppb.New(req.EndDate),
		DepartmentIds: req.DepartmentIDs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_oee_trend", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeOeeTrendResponse, error) {
			return m.coreClient.AnalyzeOeeTrend(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AnalyzeOeeTrendPresenter(resp), nil
}

func (m *analyticsSvcImpl) AnalyzeWeeksOfSales(ctx context.Context, req *AnalyzeWeeksOfSalesRequest) (*apiresource.AnalyzeWeeksOfSalesResponse, *apierror.APIError) {
	pbReq := &pb.AnalyzeWeeksOfSalesRequest{}
	if req.PeriodInWeeks != nil {
		pbReq.PeriodInWeeks = req.PeriodInWeeks
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_weeks_of_sales", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeWeeksOfSalesResponse, error) {
			return m.coreClient.AnalyzeWeeksOfSales(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AnalyzeWeeksOfSalesPresenter(resp), nil
}

func (m *analyticsSvcImpl) AnalyzeScheduleAttainment(ctx context.Context, req *AnalyzeScheduleAttainmentRequest) (*apiresource.AnalyzeScheduleAttainmentResponse, *apierror.APIError) {
	pbReq := &pb.AnalyzeScheduleAttainmentRequest{
		StartDate:     timestamppb.New(req.StartDate),
		EndDate:       timestamppb.New(req.EndDate),
		GroupBy:       groupByOrDefault(req.GroupBy),
		MachineIds:    req.MachineIDs,
		DepartmentIds: req.DepartmentIDs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_schedule_attainment", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeScheduleAttainmentResponse, error) {
			return m.coreClient.AnalyzeScheduleAttainment(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return scheduleAttainmentFromProto(resp), nil
}

func attainmentBucketFromProto(b *pb.AttainmentBucketInfo) apiresource.AttainmentBucket {
	if b == nil {
		return apiresource.AttainmentBucket{}
	}
	bucket := apiresource.AttainmentBucket{
		Key:               b.Key,
		Label:             b.Label,
		PlannedQuantity:   b.PlannedQuantity,
		ActualQuantity:    b.ActualQuantity,
		MatchedQuantity:   b.MatchedQuantity,
		WasteQuantity:     b.WasteQuantity,
		UnplannedQuantity: b.UnplannedQuantity,
		PlannedRunHours:   b.PlannedRunHours,
		PlannedLines:      b.PlannedLines,
		BatchCount:        b.BatchCount,
		// Carried through as pointers so "nothing was planned" stays distinguishable from "nothing was achieved" all the way to the client.
		AttainmentPct:  b.AttainmentPct,
		OutputRatioPct: b.OutputRatioPct,
	}
	bucket.WeekStartDate = grpcutil.TimestampToTimePtr(b.WeekStartDate)
	return bucket
}

func scheduleAttainmentFromProto(resp *pb.AnalyzeScheduleAttainmentResponse) *apiresource.AnalyzeScheduleAttainmentResponse {
	buckets := make([]apiresource.AttainmentBucket, len(resp.Buckets))
	for i, b := range resp.Buckets {
		buckets[i] = attainmentBucketFromProto(b)
	}

	adherence := make([]apiresource.FrozenAdherence, len(resp.FrozenAdherence))
	for i, a := range resp.FrozenAdherence {
		entry := apiresource.FrozenAdherence{
			Schedule:              apiresource.NewEntity(a.ScheduleId, constants.ObjectTypeProductionSchedule, nil, nil),
			Version:               a.Version,
			FrozenLineCount:       a.FrozenLineCount,
			FrozenPlannedQuantity: a.FrozenPlannedQuantity,
			DeviatedLines:         a.DeviatedLines,
			AddedLines:            a.AddedLines,
			AbsDeltaUnits:         a.AbsDeltaUnits,
			LineAdherencePct:      a.LineAdherencePct,
			UnitsAdherencePct:     a.UnitsAdherencePct,
			OffPlanLines:          a.OffPlanLines,
			OffPlanQuantity:       a.OffPlanQuantity,
		}
		entry.FrozenThroughDate = grpcutil.TimestampToTimePtr(a.FrozenThroughDate)
		adherence[i] = entry
	}

	baselineSchedules := make([]apiresource.Entity, len(resp.BaselineScheduleIds))
	for i, scheduleID := range resp.BaselineScheduleIds {
		baselineSchedules[i] = *apiresource.NewEntity(scheduleID, constants.ObjectTypeProductionSchedule, nil, nil)
	}

	baselineStatus := constants.AttainmentBaselineStatusNoBaseline
	if resp.HasBaseline {
		baselineStatus = constants.AttainmentBaselineStatusMeasured
	}

	return &apiresource.AnalyzeScheduleAttainmentResponse{
		Object:                constants.ObjectTypeAnalyzeScheduleAttainmentResponse,
		StartDate:             grpcutil.TimestampToTime(resp.StartDate),
		EndDate:               grpcutil.TimestampToTime(resp.EndDate),
		GroupBy:               constants.AttainmentGroupBy(resp.GroupBy),
		BaselineSchedules:     apiresource.NewList(baselineSchedules, apiresource.PageInfo{}),
		Buckets:               apiresource.NewList(buckets, apiresource.PageInfo{}),
		Totals:                attainmentBucketFromProto(resp.Totals),
		FrozenAdherence:       apiresource.NewList(adherence, apiresource.PageInfo{}),
		BaselineStatus:        baselineStatus,
		ScheduledMachineCount: resp.ScheduledMachineCount,
	}
}

// groupByOrDefault resolves the optional grouping. The default is documented on the endpoint, so an omitted value must not read as an invalid one.
func groupByOrDefault(groupBy field.Optional[constants.AttainmentGroupBy]) string {
	if value, ok := groupBy.Value(); ok {
		return string(value)
	}
	return string(constants.AttainmentGroupByWeek)
}

// deliveryPerformanceFromProto maps one period's delivery figures onto the API shape.
func deliveryPerformanceFromProto(p *pb.DeliveryPerformanceProto) apiresource.DeliveryPerformance {
	out := apiresource.DeliveryPerformance{
		Object:                       constants.ObjectTypeDeliveryPerformance,
		CommittedOrderCount:          p.CommittedOrderCount,
		ShippedOrderCount:            p.ShippedOrderCount,
		OnTimeOrderCount:             p.OnTimeOrderCount,
		OnTimeInFullCount:            p.OnTimeInFullCount,
		LateOrderCount:               p.LateOrderCount,
		NotYetShippedCount:           p.NotYetShippedCount,
		OnTimePct:                    p.OnTimePct,
		OnTimeInFullPct:              p.OnTimeInFullPct,
		AverageDaysLate:              p.AverageDaysLate,
		AverageLeadTimeDays:          p.AverageLeadTimeDays,
		AverageCommittedLeadTimeDays: p.AverageCommittedLeadTimeDays,
	}
	if p.PeriodStart != nil {
		t := grpcutil.TimestampToTime(p.PeriodStart)
		out.PeriodStart = &t
	}
	return out
}

// deliveryBreakdownsFromProto maps one dimension's rows, preserving the worst-first order core-service sorted them into.
func deliveryBreakdownsFromProto(rows []*pb.DeliveryBreakdownProto) *apiresource.List[apiresource.DeliveryBreakdown] {
	out := make([]apiresource.DeliveryBreakdown, 0, len(rows))
	for _, row := range rows {
		entry := apiresource.DeliveryBreakdown{
			Object: constants.ObjectTypeDeliveryBreakdown,
			Key:    row.Key,
			Label:  row.Label,
		}
		if row.Performance != nil {
			performance := deliveryPerformanceFromProto(row.Performance)
			entry.Performance = &performance
		}
		out = append(out, entry)
	}
	return apiresource.NewList(out, apiresource.PageInfo{})
}

func (m *analyticsSvcImpl) AnalyzeDeliveryPerformance(ctx context.Context, req *AnalyzeDeliveryPerformanceRequest) (*apiresource.AnalyzeDeliveryPerformanceResponse, *apierror.APIError) {
	pbReq := &pb.AnalyzeDeliveryPerformanceRequest{
		StartsAt:         timestamppb.New(req.StartDate),
		EndsAt:           timestamppb.New(req.EndDate),
		CustomerIds:      req.CustomerIDs,
		CustomerGroupIds: req.CustomerGroupIDs,
		ProductLineIds:   req.ProductLineIDs,
		SalesRepIds:      req.SalesRepIDs,
	}
	if v, ok := req.Granularity.Value(); ok {
		pbReq.Granularity = string(v)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_delivery_performance", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeDeliveryPerformanceResponse, error) {
			return m.coreClient.AnalyzeDeliveryPerformance(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	periods := make([]apiresource.DeliveryPerformance, 0, len(resp.Periods))
	for _, p := range resp.Periods {
		periods = append(periods, deliveryPerformanceFromProto(p))
	}

	backlog := make([]apiresource.DeliveryBacklogBucket, 0, len(resp.Backlog))
	for _, b := range resp.Backlog {
		backlog = append(backlog, apiresource.DeliveryBacklogBucket{
			Object:      constants.ObjectTypeDeliveryBacklogBucket,
			Label:       b.Label,
			MinDaysLate: b.MinDaysLate,
			MaxDaysLate: b.MaxDaysLate,
			OrderCount:  b.OrderCount,
			Units:       b.Units,
		})
	}

	lateness := make([]apiresource.DeliveryLatenessBucket, 0, len(resp.Lateness))
	for _, b := range resp.Lateness {
		lateness = append(lateness, apiresource.DeliveryLatenessBucket{
			Object:       constants.ObjectTypeDeliveryLatenessBucket,
			Label:        b.Label,
			MinDaysLate:  b.MinDaysLate,
			MaxDaysLate:  b.MaxDaysLate,
			OrderCount:   b.OrderCount,
			ShippedCount: b.ShippedCount,
			Units:        b.Units,
		})
	}

	out := &apiresource.AnalyzeDeliveryPerformanceResponse{
		Object:                constants.ObjectTypeAnalyzeDeliveryPerformanceResponse,
		Periods:               apiresource.NewList(periods, apiresource.PageInfo{}),
		Backlog:               apiresource.NewList(backlog, apiresource.PageInfo{}),
		Lateness:              apiresource.NewList(lateness, apiresource.PageInfo{}),
		ByCustomer:            deliveryBreakdownsFromProto(resp.ByCustomer),
		ByCustomerGroup:       deliveryBreakdownsFromProto(resp.ByCustomerGroup),
		ByProductLine:         deliveryBreakdownsFromProto(resp.ByProductLine),
		ByCommitmentSource:    deliveryBreakdownsFromProto(resp.ByCommitmentSource),
		UncommittedOrderCount: resp.UncommittedOrderCount,
	}
	if resp.Overall != nil {
		overall := deliveryPerformanceFromProto(resp.Overall)
		out.Overall = &overall
	}
	return out, nil
}
