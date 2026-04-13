package addressep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AddressSvc interface {
	ListAddresses(ctx context.Context, req *ListAddressesRequest) (*apiresource.List[apiresource.Address], *apierror.APIError)
	GetAddress(ctx context.Context, req *GetAddressRequest) (*apiresource.Address, *apierror.APIError)
	CreateAddress(ctx context.Context, req *apirequest.AddressInput) (*apiresource.Address, *apierror.APIError)
	UpdateAddress(ctx context.Context, req *UpdateAddressRequest) (*apiresource.Address, *apierror.APIError)
	DeleteAddress(ctx context.Context, req *DeleteAddressRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type AddressSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type addressSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var addressSvcTracer = tracing.GetTracer("api-gateway.endpoints.addresses.service")

func (c *AddressSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("address endpoint service: core client is required")
	}
	return nil
}

func NewAddressSvc(config *AddressSvcConfig) AddressSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &addressSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *addressSvcImpl) ListAddresses(ctx context.Context, req *ListAddressesRequest) (*apiresource.List[apiresource.Address], *apierror.APIError) {
	pbReq := &pb.ListAddressesRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, addressSvcTracer, "service.addresses.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAddressesResponse, error) {
			return m.coreClient.ListAddresses(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AddressListPresenter(resp), nil
}

func (m *addressSvcImpl) GetAddress(ctx context.Context, req *GetAddressRequest) (*apiresource.Address, *apierror.APIError) {
	pbReq := &pb.GetAddressRequest{
		Id: req.AddressID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, addressSvcTracer, "service.addresses.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAddressResponse, error) {
			return m.coreClient.GetAddress(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := AddressPresenter(resp.Address)
	return &result, nil
}

func (m *addressSvcImpl) CreateAddress(ctx context.Context, req *apirequest.AddressInput) (*apiresource.Address, *apierror.APIError) {
	pbReq := &pb.CreateAddressRequest{
		Name:         req.Name,
		Phone:        req.Phone,
		Email:        req.Email,
		IsDropShip:   req.IsDropShip,
		StreetLine_1: req.StreetLine1,
		StreetLine_2: req.StreetLine2,
		Locality:     req.Locality,
		State:        req.State,
		PostalCode:   req.PostalCode,
		Country:      req.Country,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, addressSvcTracer, "service.addresses.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateAddressResponse, error) {
			return m.coreClient.CreateAddress(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := AddressPresenter(resp.Address)
	return &result, nil
}

func (m *addressSvcImpl) UpdateAddress(ctx context.Context, req *UpdateAddressRequest) (*apiresource.Address, *apierror.APIError) {
	pbReq := &pb.UpdateAddressRequest{
		Id:           req.AddressID,
		Name:         req.Name,
		Phone:        req.Phone,
		Email:        req.Email,
		IsDropShip:   req.IsDropShip,
		StreetLine_1: req.StreetLine1,
		StreetLine_2: req.StreetLine2,
		Locality:     req.Locality,
		State:        req.State,
		PostalCode:   req.PostalCode,
		Country:      req.Country,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, addressSvcTracer, "service.addresses.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateAddressResponse, error) {
			return m.coreClient.UpdateAddress(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := AddressPresenter(resp.Address)
	return &result, nil
}

func (m *addressSvcImpl) DeleteAddress(ctx context.Context, req *DeleteAddressRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteAddressRequest{
		Id: req.AddressID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, addressSvcTracer, "service.addresses.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteAddress(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}
