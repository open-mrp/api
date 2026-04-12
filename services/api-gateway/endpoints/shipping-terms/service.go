package shippingtermep

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
	"google.golang.org/protobuf/types/known/emptypb"
)

type ShippingTermSvc interface {
	ListShippingTerms(ctx context.Context, req *ListShippingTermsRequest) (*apiresource.List[apiresource.ShippingTerm], *apierror.APIError)
	GetShippingTerm(ctx context.Context, req *GetShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError)
	CreateShippingTerm(ctx context.Context, req *CreateShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError)
	UpdateShippingTerm(ctx context.Context, req *UpdateShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError)
	DeleteShippingTerm(ctx context.Context, req *DeleteShippingTermRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type ShippingTermSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type shippingTermSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var shippingTermSvcTracer = tracing.GetTracer("api-gateway.endpoints.shipping-terms.service")

func (c *ShippingTermSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("shipping term endpoint service: core client is required")
	}
	return nil
}

func NewShippingTermSvc(config *ShippingTermSvcConfig) ShippingTermSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &shippingTermSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *shippingTermSvcImpl) ListShippingTerms(ctx context.Context, req *ListShippingTermsRequest) (*apiresource.List[apiresource.ShippingTerm], *apierror.APIError) {
	pbReq := &pb.ListShippingTermsRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shippingTermSvcTracer, "service.shipping_terms.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListShippingTermsResponse, error) {
			return m.coreClient.ListShippingTerms(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return ShippingTermListPresenter(resp), nil
}

func (m *shippingTermSvcImpl) GetShippingTerm(ctx context.Context, req *GetShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError) {
	pbReq := &pb.GetShippingTermRequest{
		Id: req.ShippingTermID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shippingTermSvcTracer, "service.shipping_terms.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetShippingTermResponse, error) {
			return m.coreClient.GetShippingTerm(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ShippingTermPresenter(resp.ShippingTerm)
	return &result, nil
}

func (m *shippingTermSvcImpl) CreateShippingTerm(ctx context.Context, req *CreateShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError) {
	pbReq := &pb.CreateShippingTermRequest{
		Name:                        req.Name,
		Type:                        string(req.Type),
		FreeShippingServiceLevelIds: req.FreeShippingServiceLevelIDs,
	}
	if req.FlatRate != nil {
		pbReq.FlatRate = &pb.QuantityInput{
			Value:  req.FlatRate.Value,
			UnitId: req.FlatRate.UnitID,
		}
	}
	if req.MinimumOrderValue != nil {
		pbReq.MinimumOrderValue = &pb.QuantityInput{
			Value:  req.MinimumOrderValue.Value,
			UnitId: req.MinimumOrderValue.UnitID,
		}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shippingTermSvcTracer, "service.shipping_terms.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateShippingTermResponse, error) {
			return m.coreClient.CreateShippingTerm(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ShippingTermPresenter(resp.ShippingTerm)
	return &result, nil
}

func (m *shippingTermSvcImpl) UpdateShippingTerm(ctx context.Context, req *UpdateShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError) {
	pbReq := &pb.UpdateShippingTermRequest{
		Id:   req.ShippingTermID,
		Name: req.Name,
	}
	if req.Type != nil {
		t := string(*req.Type)
		pbReq.Type = &t
	}
	if req.FlatRate.IsSet() {
		pbReq.HasFlatRate = true
		if !req.FlatRate.IsNull() {
			v := req.FlatRate.Value()
			pbReq.FlatRate = &pb.QuantityInput{
				Value:  v.Value,
				UnitId: v.UnitID,
			}
		}
	}
	if req.MinimumOrderValue.IsSet() {
		pbReq.HasMinimumOrderValue = true
		if !req.MinimumOrderValue.IsNull() {
			v := req.MinimumOrderValue.Value()
			pbReq.MinimumOrderValue = &pb.QuantityInput{
				Value:  v.Value,
				UnitId: v.UnitID,
			}
		}
	}
	if req.FreeShippingServiceLevelIDs.IsSet() {
		pbReq.HasFreeShippingServiceLevelIds = true
		if !req.FreeShippingServiceLevelIDs.IsNull() {
			pbReq.FreeShippingServiceLevelIds = *req.FreeShippingServiceLevelIDs.Value()
		}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shippingTermSvcTracer, "service.shipping_terms.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateShippingTermResponse, error) {
			return m.coreClient.UpdateShippingTerm(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ShippingTermPresenter(resp.ShippingTerm)
	return &result, nil
}

func (m *shippingTermSvcImpl) DeleteShippingTerm(ctx context.Context, req *DeleteShippingTermRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteShippingTermRequest{
		Id: req.ShippingTermID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, shippingTermSvcTracer, "service.shipping_terms.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteShippingTerm(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}
