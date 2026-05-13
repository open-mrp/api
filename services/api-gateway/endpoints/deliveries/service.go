package deliveryep

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
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DeliverySvc interface {
	ListDeliveries(ctx context.Context, req *ListDeliveriesRequest) (*apiresource.List[apiresource.DeliverySummary], *apierror.APIError)
	GetDelivery(ctx context.Context, req *RetrieveDeliveryRequest) (*apiresource.Delivery, *apierror.APIError)
}

type DeliverySvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type deliverySvcImpl struct {
	coreClient pb.CoreServiceClient
}

var deliverySvcTracer = tracing.GetTracer("api-gateway.endpoints.deliveries.service")

func (c *DeliverySvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("delivery endpoint service: core client is required")
	}
	return nil
}

func NewDeliverySvc(config *DeliverySvcConfig) DeliverySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &deliverySvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *deliverySvcImpl) ListDeliveries(ctx context.Context, req *ListDeliveriesRequest) (*apiresource.List[apiresource.DeliverySummary], *apierror.APIError) {
	pbReq := &pb.ListDeliveriesRequest{
		Cursor:      req.Cursor,
		Limit:       req.Limit,
		Query:       req.Query,
		Status:      req.Status,
		ItemIds:     req.ItemIDs,
		SupplierIds: req.SupplierIDs,
	}

	if req.StartDate != nil {
		t, err := grpcutil.ParseDateString(*req.StartDate)
		if err == nil {
			pbReq.StartDate = timestamppb.New(t)
		}
	}
	if req.EndDate != nil {
		t, err := grpcutil.ParseDateString(*req.EndDate)
		if err == nil {
			pbReq.EndDate = timestamppb.New(t)
		}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, deliverySvcTracer, "service.deliveries.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListDeliveriesResponse, error) {
			return m.coreClient.ListDeliveries(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return DeliveryListPresenter(resp), nil
}

func (m *deliverySvcImpl) GetDelivery(ctx context.Context, req *RetrieveDeliveryRequest) (*apiresource.Delivery, *apierror.APIError) {
	pbReq := &pb.GetDeliveryRequest{
		Id: req.DeliveryID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, deliverySvcTracer, "service.deliveries.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetDeliveryResponse, error) {
			return m.coreClient.GetDelivery(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := DeliveryPresenter(resp.Delivery)
	return &result, nil
}
