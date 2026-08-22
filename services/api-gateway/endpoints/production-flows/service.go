package productionflowep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProductionFlowSvc interface {
	GetProductionFlow(ctx context.Context, req *GetProductionFlowRequest) (*apiresource.ProductionFlow, *apierror.APIError)
	ConnectSteps(ctx context.Context, req *ConnectStepsRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type ProductionFlowSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type productionFlowSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var productionFlowSvcTracer = tracing.GetTracer("api-gateway.endpoints.production-flows.service")

func (c *ProductionFlowSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("production flow endpoint service: core client is required")
	}
	return nil
}

func NewProductionFlowSvc(config *ProductionFlowSvcConfig) ProductionFlowSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &productionFlowSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *productionFlowSvcImpl) GetProductionFlow(ctx context.Context, req *GetProductionFlowRequest) (*apiresource.ProductionFlow, *apierror.APIError) {
	pbReq := &pb.GetProductionFlowRequest{
		ItemId: req.ItemID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionFlowSvcTracer, "service.production_flows.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetProductionFlowResponse, error) {
			return m.coreClient.GetProductionFlow(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ProductionFlowPresenter(ctx, resp.Steps)
	return result, nil
}

func (m *productionFlowSvcImpl) ConnectSteps(ctx context.Context, req *ConnectStepsRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.ConnectProductionStepsRequest{
		SourceProductionStepId: req.SourceProductionStepID,
		TargetProductionStepId: req.TargetProductionStepID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, productionFlowSvcTracer, "service.production_flows.connect_steps", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.ConnectProductionSteps(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}
