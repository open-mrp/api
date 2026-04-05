package shippingcaseep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type ShippingCaseSvc interface {
	GetShippingCase(ctx context.Context, req *GetShippingCaseRequest) (*apiresource.ShippingCase, *apierror.APIError)
	UpdateShippingCase(ctx context.Context, req *UpdateShippingCaseRequest) (*apiresource.ShippingCase, *apierror.APIError)
	DeleteShippingCase(ctx context.Context, req *DeleteShippingCaseRequest) (*apiresource.EmptyResource, *apierror.APIError)
	GetShippingCaseLabel(ctx context.Context, req *GetShippingCaseLabelRequest) (*apiresource.ShippingCaseLabelURL, *apierror.APIError)
}

type ShippingCaseSvcConfig struct {
	CoreClient pb.CoreShippingCaseServiceClient
}

type shippingCaseSvcImpl struct {
	coreClient pb.CoreShippingCaseServiceClient
}

var shippingCaseSvcTracer = tracing.GetTracer("api-gateway.endpoints.shipping-cases.service")

func (c *ShippingCaseSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("shipping case endpoint service: core client is required")
	}
	return nil
}

func NewShippingCaseSvc(config *ShippingCaseSvcConfig) ShippingCaseSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &shippingCaseSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *shippingCaseSvcImpl) GetShippingCase(ctx context.Context, req *GetShippingCaseRequest) (*apiresource.ShippingCase, *apierror.APIError) {
	pbReq := &pb.GetShippingCaseRequest{
		Id: req.ShippingCaseID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shippingCaseSvcTracer, "service.shipping_cases.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetShippingCaseResponse, error) {
			return m.coreClient.GetShippingCase(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ShippingCasePresenter(resp.ShippingCase)
	return &result, nil
}

func (m *shippingCaseSvcImpl) UpdateShippingCase(ctx context.Context, req *UpdateShippingCaseRequest) (*apiresource.ShippingCase, *apierror.APIError) {
	pbReq := &pb.UpdateShippingCaseRequest{
		Id:                  req.ShippingCaseID,
		TrackingNumber:      req.TrackingNumber,
		FreightAmountValue:  req.FreightAmountValue,
		FreightAmountUnitId: req.FreightAmountUnitID,
		FreightWeightValue:  req.FreightWeightValue,
		FreightWeightUnitId: req.FreightWeightUnitID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shippingCaseSvcTracer, "service.shipping_cases.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateShippingCaseResponse, error) {
			return m.coreClient.UpdateShippingCase(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ShippingCasePresenter(resp.ShippingCase)
	return &result, nil
}

func (m *shippingCaseSvcImpl) DeleteShippingCase(ctx context.Context, req *DeleteShippingCaseRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteShippingCaseRequest{
		Id: req.ShippingCaseID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, shippingCaseSvcTracer, "service.shipping_cases.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteShippingCaseResponse, error) {
			return m.coreClient.DeleteShippingCase(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *shippingCaseSvcImpl) GetShippingCaseLabel(ctx context.Context, req *GetShippingCaseLabelRequest) (*apiresource.ShippingCaseLabelURL, *apierror.APIError) {
	pbReq := &pb.GetShippingCaseLabelRequest{
		Id: req.ShippingCaseID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shippingCaseSvcTracer, "service.shipping_cases.get_label", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetShippingCaseLabelResponse, error) {
			return m.coreClient.GetShippingCaseLabel(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.ShippingCaseLabelURL{
		Object: constants.ObjectTypeShippingCaseLabelURL,
		URL:    resp.Url,
	}, nil
}
