package addressvalidationep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

type AddressValidationSvc interface {
	AutocompleteAddress(ctx context.Context, req *AutocompleteAddressRequest) (*apiresource.List[apiresource.AddressSuggestion], *apierror.APIError)
	GetAddressDetails(ctx context.Context, req *RetrieveAddressDetailsRequest) (*apiresource.AddressDetailsResult, *apierror.APIError)
	ValidateAddress(ctx context.Context, req *ValidateAddressRequest) (*apiresource.ValidatedAddress, *apierror.APIError)
}

type AddressValidationSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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

	return suggestionListFromProto(resp), nil
}

func (m *addressValidationSvcImpl) GetAddressDetails(ctx context.Context, req *RetrieveAddressDetailsRequest) (*apiresource.AddressDetailsResult, *apierror.APIError) {
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

	return addressDetailsFromProto(resp), nil
}

func (m *addressValidationSvcImpl) ValidateAddress(ctx context.Context, req *ValidateAddressRequest) (*apiresource.ValidatedAddress, *apierror.APIError) {
	pbReq := &pb.ValidateAddressRequest{
		AddressLine_1: req.AddressLine1,
		AddressLine_2: req.AddressLine2.Ptr(),
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

	return validatedAddressFromProto(resp), nil
}

func suggestionFromProto(s *pb.AddressSuggestion) apiresource.AddressSuggestion {
	if s == nil {
		return apiresource.AddressSuggestion{}
	}
	return apiresource.AddressSuggestion{
		ID:            s.Id,
		Object:        constants.ObjectTypeAddressSuggestion,
		Description:   s.Description,
		MainText:      s.MainText,
		SecondaryText: s.SecondaryText,
	}
}

func suggestionListFromProto(resp *pb.AutocompleteAddressResponse) *apiresource.List[apiresource.AddressSuggestion] {
	if resp == nil {
		return apiresource.NewList[apiresource.AddressSuggestion](nil, apiresource.PageInfo{})
	}

	suggestions := make([]apiresource.AddressSuggestion, len(resp.Suggestions))
	for i, s := range resp.Suggestions {
		suggestions[i] = suggestionFromProto(s)
	}

	return apiresource.NewList(suggestions, apiresource.PageInfo{})
}

func addressComponentsFromProto(c *pb.AddressComponentsInfo) *apiresource.AddressComponents {
	if c == nil {
		return nil
	}
	return &apiresource.AddressComponents{
		Object:       constants.ObjectTypeAddressComponents,
		AddressLine1: c.AddressLine_1,
		AddressLine2: c.AddressLine_2,
		City:         c.City,
		State:        c.State,
		PostalCode:   c.PostalCode,
		Country:      c.Country,
		CountryCode:  c.CountryCode,
	}
}

func addressDetailsFromProto(resp *pb.GetAddressDetailsResponse) *apiresource.AddressDetailsResult {
	if resp == nil {
		return nil
	}
	return &apiresource.AddressDetailsResult{
		Object:           constants.ObjectTypeAddressDetailsResult,
		Address:          addressComponentsFromProto(resp.Address),
		FormattedAddress: resp.FormattedAddress,
	}
}

func validatedAddressFromProto(resp *pb.ValidateAddressResponse) *apiresource.ValidatedAddress {
	if resp == nil {
		return nil
	}

	validationMessages := resp.ValidationMessages
	if validationMessages == nil {
		validationMessages = []string{}
	}

	return &apiresource.ValidatedAddress{
		Object:             constants.ObjectTypeValidatedAddress,
		Status:             addressValidationStatus(resp.IsValid),
		FormattedAddress:   resp.FormattedAddress,
		Components:         addressComponentsFromProto(resp.Components),
		ValidationMessages: validationMessages,
	}
}

func addressValidationStatus(isValid bool) constants.AddressValidationStatus {
	if isValid {
		return constants.AddressValidationStatusValid
	}
	return constants.AddressValidationStatusInvalid
}
