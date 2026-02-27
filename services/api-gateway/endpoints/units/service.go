package unitep

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

type UnitSvc interface {
	ListUnits(ctx context.Context, req *ListUnitsRequest) (*apiresource.List[apiresource.Unit], *apierror.APIError)
}

type UnitSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type unitSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var unitSvcTracer = tracing.GetTracer("api-gateway.endpoints.units.service")

func (c *UnitSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("unit endpoint service: core client is required")
	}
	return nil
}

func NewUnitSvc(config *UnitSvcConfig) UnitSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &unitSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *unitSvcImpl) ListUnits(ctx context.Context, req *ListUnitsRequest) (*apiresource.List[apiresource.Unit], *apierror.APIError) {
	pbReq := &pb.ListUnitsRequest{
		Cursor:       req.Cursor,
		Limit:        req.Limit,
		Query:        req.Query,
		Type:         req.Type,
		UnitGroupIds: req.UnitGroupIDs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, unitSvcTracer, "service.units.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListUnitsResponse, error) {
			return m.coreClient.ListUnits(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return UnitListPresenter(resp), nil
}
