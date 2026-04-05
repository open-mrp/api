package rateep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type RateSvc interface {
	UpdateRate(ctx context.Context, req *UpdateRateRequest) (*apiresource.Rate, *apierror.APIError)
}

type RateSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type rateSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var rateSvcTracer = tracing.GetTracer("api-gateway.endpoints.rates.service")

func (c *RateSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("rate endpoint service: core client is required")
	}
	return nil
}

func NewRateSvc(config *RateSvcConfig) RateSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &rateSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *rateSvcImpl) UpdateRate(ctx context.Context, req *UpdateRateRequest) (*apiresource.Rate, *apierror.APIError) {
	pbReq := &pb.UpdateRateRequest{
		Id:                req.RateID,
		Value:             req.Value,
		NumeratorUnitId:   req.NumeratorUnitID,
		DenominatorUnitId: req.DenominatorUnitID,
		ObjectId:          req.ObjectID,
		ObjectType:        req.ObjectType,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, rateSvcTracer, "service.rates.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateRateResponse, error) {
			return m.coreClient.UpdateRate(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := RatePresenter(resp.Rate)
	return &result, nil
}
