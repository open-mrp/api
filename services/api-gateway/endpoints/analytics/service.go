package analyticsep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	"github.com/augno/api/shared/safeconv"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
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
	AnalyzeWeeksOfSales(ctx context.Context, req *AnalyzeWeeksOfSalesRequest) (*apiresource.AnalyzeWeeksOfSalesResponse, *apierror.APIError)
}

type AnalyticsSvcConfig struct {
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
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok && identity != nil && identity.Actor != nil && identity.Actor.RoleTypeCode != nil {
		isSalesRep = *identity.Actor.RoleTypeCode == string(constants.RoleTypeCodeSalesRep)
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
	pbReq := &pb.AnalyzeOeeRequest{
		StartDate:     timestamppb.New(req.StartDate),
		EndDate:       timestamppb.New(req.EndDate),
		DepartmentIds: req.DepartmentIDs,
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
