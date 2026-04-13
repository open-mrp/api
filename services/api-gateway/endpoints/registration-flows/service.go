package registrationflowep

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

type RegistrationFlowSvc interface {
	ListRegistrationFlows(ctx context.Context, req *ListRegistrationFlowsRequest) (*apiresource.List[apiresource.RegistrationFlow], *apierror.APIError)
	GetRegistrationFlow(ctx context.Context, req *GetRegistrationFlowRequest) (*apiresource.RegistrationFlow, *apierror.APIError)
	CreateRegistrationFlow(ctx context.Context, req *CreateRegistrationFlowRequest) (*apiresource.RegistrationFlow, *apierror.APIError)
	UpdateRegistrationFlow(ctx context.Context, req *UpdateRegistrationFlowRequest) (*apiresource.RegistrationFlow, *apierror.APIError)
	DeleteRegistrationFlow(ctx context.Context, req *DeleteRegistrationFlowRequest) (*apiresource.EmptyResource, *apierror.APIError)
	GetRegistrationFlowBySlug(ctx context.Context, req *GetRegistrationFlowBySlugRequest) (*apiresource.RegistrationFlow, *apierror.APIError)
	RegisterCustomer(ctx context.Context, req *RegisterCustomerRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type RegistrationFlowSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type registrationFlowSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var registrationFlowSvcTracer = tracing.GetTracer("api-gateway.endpoints.registration-flows.service")

func (c *RegistrationFlowSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("registration flow endpoint service: core client is required")
	}
	return nil
}

func NewRegistrationFlowSvc(config *RegistrationFlowSvcConfig) RegistrationFlowSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &registrationFlowSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *registrationFlowSvcImpl) ListRegistrationFlows(ctx context.Context, req *ListRegistrationFlowsRequest) (*apiresource.List[apiresource.RegistrationFlow], *apierror.APIError) {
	pbReq := &pb.ListRegistrationFlowsRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, registrationFlowSvcTracer, "service.registration_flows.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListRegistrationFlowsResponse, error) {
			return m.coreClient.ListRegistrationFlows(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return RegistrationFlowListPresenter(resp), nil
}

func (m *registrationFlowSvcImpl) GetRegistrationFlow(ctx context.Context, req *GetRegistrationFlowRequest) (*apiresource.RegistrationFlow, *apierror.APIError) {
	pbReq := &pb.GetRegistrationFlowRequest{
		Id: req.RegistrationFlowID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, registrationFlowSvcTracer, "service.registration_flows.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetRegistrationFlowResponse, error) {
			return m.coreClient.GetRegistrationFlow(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := RegistrationFlowPresenter(resp.RegistrationFlow)
	return &result, nil
}

func (m *registrationFlowSvcImpl) CreateRegistrationFlow(ctx context.Context, req *CreateRegistrationFlowRequest) (*apiresource.RegistrationFlow, *apierror.APIError) {
	pbReq := &pb.CreateRegistrationFlowRequest{
		Name:             req.Name,
		CustomerGroupIds: req.CustomerGroupIDs,
		PaymentTermIds:   req.PaymentTermIDs,
		ShippingTermIds:  req.ShippingTermIDs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, registrationFlowSvcTracer, "service.registration_flows.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateRegistrationFlowResponse, error) {
			return m.coreClient.CreateRegistrationFlow(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := RegistrationFlowPresenter(resp.RegistrationFlow)
	return &result, nil
}

func (m *registrationFlowSvcImpl) UpdateRegistrationFlow(ctx context.Context, req *UpdateRegistrationFlowRequest) (*apiresource.RegistrationFlow, *apierror.APIError) {
	pbReq := &pb.UpdateRegistrationFlowRequest{
		Id:                  req.RegistrationFlowID,
		Name:                req.Name,
		CustomerGroupIds:    req.CustomerGroupIDs,
		PaymentTermIds:      req.PaymentTermIDs,
		ShippingTermIds:     req.ShippingTermIDs,
		HasCustomerGroupIds: req.HasCustomerGroupIDs,
		HasPaymentTermIds:   req.HasPaymentTermIDs,
		HasShippingTermIds:  req.HasShippingTermIDs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, registrationFlowSvcTracer, "service.registration_flows.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateRegistrationFlowResponse, error) {
			return m.coreClient.UpdateRegistrationFlow(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := RegistrationFlowPresenter(resp.RegistrationFlow)
	return &result, nil
}

func (m *registrationFlowSvcImpl) DeleteRegistrationFlow(ctx context.Context, req *DeleteRegistrationFlowRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteRegistrationFlowRequest{
		Id: req.RegistrationFlowID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, registrationFlowSvcTracer, "service.registration_flows.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteRegistrationFlow(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *registrationFlowSvcImpl) GetRegistrationFlowBySlug(ctx context.Context, req *GetRegistrationFlowBySlugRequest) (*apiresource.RegistrationFlow, *apierror.APIError) {
	pbReq := &pb.GetRegistrationFlowBySlugRequest{
		Slug: req.Slug,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, registrationFlowSvcTracer, "service.registration_flows.get_by_slug", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetRegistrationFlowBySlugResponse, error) {
			return m.coreClient.GetRegistrationFlowBySlug(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := RegistrationFlowPresenter(resp.RegistrationFlow)
	return &result, nil
}

func (m *registrationFlowSvcImpl) RegisterCustomer(ctx context.Context, req *RegisterCustomerRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.RegisterCustomerRequest{
		AccountSlug:        req.AccountSlug,
		IsExistingCustomer: req.IsExistingCustomer,
		CustomerNumber:     req.CustomerNumber,
		CustomerName:       req.CustomerName,
		CustomerGroupId:    req.CustomerGroupID,
		Phone:              req.Phone,
		ShippingTermId:     req.ShippingTermID,
		PaymentTermId:      req.PaymentTermID,
	}

	if req.Address != nil {
		pbReq.Address = &pb.RegisterCustomerAddressInput{
			StreetLine_1: derefStr(req.Address.StreetLine1),
			StreetLine_2: req.Address.StreetLine2,
			Locality:     derefStr(req.Address.Locality),
			State:        derefStr(req.Address.State),
			PostalCode:   derefStr(req.Address.PostalCode),
			Country:      req.Address.Country,
			Name:         &req.Address.Name,
		}
	}

	_, apiErr := grpcutil.CallRPC(ctx, registrationFlowSvcTracer, "service.registration_flows.register_customer", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.RegisterCustomerResponse, error) {
			return m.coreClient.RegisterCustomer(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
