package quantityep

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

type QuantitySvc interface {
	UpdateQuantity(ctx context.Context, req *UpdateQuantityRequest) (*apiresource.Quantity, *apierror.APIError)
}

type QuantitySvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type quantitySvcImpl struct {
	coreClient pb.CoreServiceClient
}

var quantitySvcTracer = tracing.GetTracer("api-gateway.endpoints.quantities.service")

func (c *QuantitySvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("quantity endpoint service: core client is required")
	}
	return nil
}

func NewQuantitySvc(config *QuantitySvcConfig) QuantitySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &quantitySvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *quantitySvcImpl) UpdateQuantity(ctx context.Context, req *UpdateQuantityRequest) (*apiresource.Quantity, *apierror.APIError) {
	pbReq := &pb.UpdateQuantityRequest{
		Id:         req.QuantityID,
		Value:      req.Value,
		UnitId:     req.UnitID,
		ObjectId:   req.ObjectID,
		ObjectType: req.ObjectType,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, quantitySvcTracer, "service.quantities.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateQuantityResponse, error) {
			return m.coreClient.UpdateQuantity(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := QuantityPresenter(resp.Quantity)
	return &result, nil
}
