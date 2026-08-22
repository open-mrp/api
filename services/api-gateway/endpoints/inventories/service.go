package inventoryep

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
)

type InventorySvc interface {
	ListInventories(ctx context.Context, req *ListInventoriesRequest) (*apiresource.List[apiresource.InventoryItem], *apierror.APIError)
}

type InventorySvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type inventorySvcImpl struct {
	coreClient pb.CoreServiceClient
}

var inventorySvcTracer = tracing.GetTracer("api-gateway.endpoints.inventories.service")

func (c *InventorySvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("inventory endpoint service: core client is required")
	}
	return nil
}

func NewInventorySvc(config *InventorySvcConfig) InventorySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &inventorySvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *inventorySvcImpl) ListInventories(ctx context.Context, req *ListInventoriesRequest) (*apiresource.List[apiresource.InventoryItem], *apierror.APIError) {
	pbReq := &pb.ListInventoriesRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, inventorySvcTracer, "service.inventories.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListInventoriesResponse, error) {
			return m.coreClient.ListInventories(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return ListInventoriesPresenter(ctx, resp), nil
}
