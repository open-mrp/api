package addressvalidationep

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

type AddressValidationSvc interface {
	AutocompleteAddress(ctx context.Context, req *AutocompleteAddressRequest) (*apiresource.List[apiresource.AddressSuggestion], *apierror.APIError)
	GetAddressDetails(ctx context.Context, req *GetAddressDetailsRequest) (*apiresource.AddressDetailsResult, *apierror.APIError)
	ValidateAddress(ctx context.Context, req *ValidateAddressRequest) (*apiresource.ValidatedAddress, *apierror.APIError)
}

type AddressValidationSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type addressValidationSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var addressValidationSvcTracer = tracing.GetTracer("api-gateway.endpoints.address-validation.service")

func (c *AddressValidationSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("address validation endpoint service: core client is required")
	}
	return nil
}

func NewAddressValidationSvc(config *AddressValidationSvcConfig) AddressValidationSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &addressValidationSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *addressValidationSvcImpl) AutocompleteAddress(ctx context.Context, req *AutocompleteAddressRequest) (*apiresource.List[apiresource.AddressSuggestion], *apierror.APIError) {
	pbReq := &pb.AutocompleteAddressRequest{
		Input:        req.Input,
		SessionToken: req.SessionToken,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, addressValidationSvcTracer, "service.address_validation.autocomplete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AutocompleteAddressResponse, error) {
			return m.coreClient.AutocompleteAddress(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return SuggestionListPresenter(resp), nil
}

func (m *addressValidationSvcImpl) GetAddressDetails(ctx context.Context, req *GetAddressDetailsRequest) (*apiresource.AddressDetailsResult, *apierror.APIError) {
	pbReq := &pb.GetAddressDetailsRequest{
		PlaceId:      req.PlaceID,
		SessionToken: req.SessionToken,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, addressValidationSvcTracer, "service.address_validation.details", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAddressDetailsResponse, error) {
			return m.coreClient.GetAddressDetails(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AddressDetailsPresenter(resp), nil
}

func (m *addressValidationSvcImpl) ValidateAddress(ctx context.Context, req *ValidateAddressRequest) (*apiresource.ValidatedAddress, *apierror.APIError) {
	pbReq := &pb.ValidateAddressRequest{
		AddressLine_1: req.AddressLine1,
		AddressLine_2: req.AddressLine2,
		City:          req.City,
		State:         req.State,
		PostalCode:    req.PostalCode,
		Country:       req.Country,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, addressValidationSvcTracer, "service.address_validation.validate", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ValidateAddressResponse, error) {
			return m.coreClient.ValidateAddress(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return ValidatedAddressPresenter(resp), nil
}
