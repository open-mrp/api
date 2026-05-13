package consumptionep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type ConsumptionSvc interface {
	GetConsumption(ctx context.Context, req *RetrieveConsumptionRequest) (*apiresource.Consumption, *apierror.APIError)
	CreateConsumption(ctx context.Context, req *CreateConsumptionRequest) (*apiresource.Consumption, *apierror.APIError)
	UpdateConsumption(ctx context.Context, req *UpdateConsumptionRequest) (*apiresource.Consumption, *apierror.APIError)
	DeleteConsumption(ctx context.Context, req *DeleteConsumptionRequest) (*apiresource.Consumption, *apierror.APIError)
}

type ConsumptionSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type consumptionSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var consumptionSvcTracer = tracing.GetTracer("api-gateway.endpoints.consumptions.service")

func (c *ConsumptionSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("consumption endpoint service: core client is required")
	}
	return nil
}

func NewConsumptionSvc(config *ConsumptionSvcConfig) ConsumptionSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &consumptionSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *consumptionSvcImpl) GetConsumption(ctx context.Context, req *RetrieveConsumptionRequest) (*apiresource.Consumption, *apierror.APIError) {
	pbReq := &pb.GetConsumptionRequest{
		ProductionStepId: req.ProductionStepID,
		Id:               req.ConsumptionID,
		Includes:         appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, consumptionSvcTracer, "service.consumptions.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetConsumptionResponse, error) {
			return m.coreClient.GetConsumption(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ConsumptionPresenter(resp.Consumption)
	return &result, nil
}

func (m *consumptionSvcImpl) CreateConsumption(ctx context.Context, req *CreateConsumptionRequest) (*apiresource.Consumption, *apierror.APIError) {
	pbReq := &pb.CreateConsumptionRequest{
		ProductionStepId:    req.ProductionStepID,
		ItemId:              req.ItemID,
		QuantityValue:       req.QuantityValue,
		QuantityUnitId:      req.QuantityUnitID,
		WasteQuantityValue:  req.WasteQuantityValue,
		WasteQuantityUnitId: req.WasteQuantityUnitID,
		Instructions:        req.Instructions,
		Includes:            appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, consumptionSvcTracer, "service.consumptions.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateConsumptionResponse, error) {
			return m.coreClient.CreateConsumption(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ConsumptionPresenter(resp.Consumption)
	return &result, nil
}

func (m *consumptionSvcImpl) UpdateConsumption(ctx context.Context, req *UpdateConsumptionRequest) (*apiresource.Consumption, *apierror.APIError) {
	pbReq := &pb.UpdateConsumptionRequest{
		ProductionStepId:    req.ProductionStepID,
		Id:                  req.ConsumptionID,
		ItemId:              req.ItemID,
		QuantityValue:       req.QuantityValue,
		QuantityUnitId:      req.QuantityUnitID,
		WasteQuantityValue:  req.WasteQuantityValue,
		WasteQuantityUnitId: req.WasteQuantityUnitID,
		Instructions:        req.Instructions,
		Includes:            appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, consumptionSvcTracer, "service.consumptions.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateConsumptionResponse, error) {
			return m.coreClient.UpdateConsumption(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ConsumptionPresenter(resp.Consumption)
	return &result, nil
}

func (m *consumptionSvcImpl) DeleteConsumption(ctx context.Context, req *DeleteConsumptionRequest) (*apiresource.Consumption, *apierror.APIError) {
	pbReq := &pb.DeleteConsumptionRequest{
		ProductionStepId: req.ProductionStepID,
		Id:               req.ConsumptionID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, consumptionSvcTracer, "service.consumptions.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteConsumptionResponse, error) {
			return m.coreClient.DeleteConsumption(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ConsumptionPresenter(resp.Consumption)
	return &result, nil
}
