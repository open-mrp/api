package registrationflowep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/ptrutil"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type RegistrationFlowSvc interface {
	ListRegistrationFlows(ctx context.Context, req *ListRegistrationFlowsRequest) (*apiresource.List[apiresource.RegistrationFlow], *apierror.APIError)
	GetRegistrationFlow(ctx context.Context, req *RetrieveRegistrationFlowRequest) (*apiresource.RegistrationFlow, *apierror.APIError)
	CreateRegistrationFlow(ctx context.Context, req *CreateRegistrationFlowRequest) (*apiresource.RegistrationFlow, *apierror.APIError)
	UpdateRegistrationFlow(ctx context.Context, req *UpdateRegistrationFlowRequest) (*apiresource.RegistrationFlow, *apierror.APIError)
	DeleteRegistrationFlow(ctx context.Context, req *DeleteRegistrationFlowRequest) (*apiresource.EmptyResource, *apierror.APIError)
	GetRegistrationFlowBySlug(ctx context.Context, req *RetrieveRegistrationFlowBySlugRequest) (*apiresource.RegistrationFlow, *apierror.APIError)
	RegisterCustomer(ctx context.Context, req *RegisterCustomerRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type RegistrationFlowSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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

	return registrationFlowListFromProto(ctx, resp), nil
}

func (m *registrationFlowSvcImpl) GetRegistrationFlow(ctx context.Context, req *RetrieveRegistrationFlowRequest) (*apiresource.RegistrationFlow, *apierror.APIError) {
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

	result := registrationFlowFromProto(resp.RegistrationFlow)
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

	result := registrationFlowFromProto(resp.RegistrationFlow)
	return &result, nil
}

func (m *registrationFlowSvcImpl) UpdateRegistrationFlow(ctx context.Context, req *UpdateRegistrationFlowRequest) (*apiresource.RegistrationFlow, *apierror.APIError) {
	pbReq := &pb.UpdateRegistrationFlowRequest{
		Id:                  req.RegistrationFlowID,
		Name:                req.Name.Ptr(),
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

	result := registrationFlowFromProto(resp.RegistrationFlow)
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

func (m *registrationFlowSvcImpl) GetRegistrationFlowBySlug(ctx context.Context, req *RetrieveRegistrationFlowBySlugRequest) (*apiresource.RegistrationFlow, *apierror.APIError) {
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

	result := registrationFlowFromProto(resp.RegistrationFlow)
	return &result, nil
}

func (m *registrationFlowSvcImpl) RegisterCustomer(ctx context.Context, req *RegisterCustomerRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.RegisterCustomerRequest{
		AccountSlug:        req.AccountSlug,
		IsExistingCustomer: req.IsExistingCustomer,
		CustomerNumber:     req.CustomerNumber.Ptr(),
		CustomerName:       req.CustomerName.Ptr(),
		CustomerGroupId:    req.CustomerGroupID.Ptr(),
		Phone:              req.Phone.Ptr(),
		ShippingTermId:     req.ShippingTermID.Ptr(),
		PaymentTermId:      req.PaymentTermID.Ptr(),
	}

	if addr, ok := req.Address.Value(); ok {
		pbReq.Address = &pb.RegisterCustomerAddressInput{
			StreetLine_1: ptrutil.Deref(addr.StreetLine1.Ptr()),
			StreetLine_2: addr.StreetLine2.Ptr(),
			Locality:     ptrutil.Deref(addr.Locality.Ptr()),
			State:        ptrutil.Deref(addr.State.Ptr()),
			PostalCode:   ptrutil.Deref(addr.PostalCode.Ptr()),
			Country:      addr.Country,
			Name:         &addr.Name,
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

func registrationFlowOptionFromProto(opt *pb.RegistrationFlowOptionInfo) apiresource.RegistrationFlowOption {
	if opt == nil {
		return apiresource.RegistrationFlowOption{}
	}

	return apiresource.RegistrationFlowOption{
		ID:     opt.Id,
		Object: constants.ObjectTypeRegistrationFlowOption,
		Name:   opt.Name,
	}
}

func registrationFlowFromProto(rf *pb.RegistrationFlowInfo) apiresource.RegistrationFlow {
	if rf == nil {
		return apiresource.RegistrationFlow{}
	}

	customerGroupOptions := make([]apiresource.RegistrationFlowOption, len(rf.CustomerGroupOptions))
	for i, opt := range rf.CustomerGroupOptions {
		customerGroupOptions[i] = registrationFlowOptionFromProto(opt)
	}

	paymentTermOptions := make([]apiresource.RegistrationFlowOption, len(rf.PaymentTermOptions))
	for i, opt := range rf.PaymentTermOptions {
		paymentTermOptions[i] = registrationFlowOptionFromProto(opt)
	}

	shippingTermOptions := make([]apiresource.RegistrationFlowOption, len(rf.ShippingTermOptions))
	for i, opt := range rf.ShippingTermOptions {
		shippingTermOptions[i] = registrationFlowOptionFromProto(opt)
	}

	return apiresource.RegistrationFlow{
		ID:                   rf.Id,
		Object:               constants.ObjectTypeRegistrationFlow,
		Name:                 rf.Name,
		CustomerGroupOptions: apiresource.NewList(customerGroupOptions, apiresource.PageInfo{}),
		PaymentTermOptions:   apiresource.NewList(paymentTermOptions, apiresource.PageInfo{}),
		ShippingTermOptions:  apiresource.NewList(shippingTermOptions, apiresource.PageInfo{}),
		CreatedAt:            grpcutil.TimestampToTime(rf.CreatedAt),
		UpdatedAt:            grpcutil.TimestampToTime(rf.UpdatedAt),
	}
}

func registrationFlowListFromProto(ctx context.Context, resp *pb.ListRegistrationFlowsResponse) *apiresource.List[apiresource.RegistrationFlow] {
	if resp == nil {
		return apiresource.NewList[apiresource.RegistrationFlow](nil, apiresource.PageInfo{})
	}

	registrationFlows := make([]apiresource.RegistrationFlow, len(resp.RegistrationFlows))
	for i, rf := range resp.RegistrationFlows {
		registrationFlows[i] = registrationFlowFromProto(rf)
	}

	return apiresource.NewList(registrationFlows, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
