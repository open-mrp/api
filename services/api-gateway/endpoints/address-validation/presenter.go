package addressvalidationep

import (
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func SuggestionPresenter(s *pb.AddressSuggestion) apiresource.AddressSuggestion {
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

func SuggestionListPresenter(resp *pb.AutocompleteAddressResponse) *apiresource.List[apiresource.AddressSuggestion] {
	if resp == nil {
		return apiresource.NewList[apiresource.AddressSuggestion](nil, apiresource.PageInfo{})
	}

	suggestions := make([]apiresource.AddressSuggestion, len(resp.Suggestions))
	for i, s := range resp.Suggestions {
		suggestions[i] = SuggestionPresenter(s)
	}

	return apiresource.NewList(suggestions, apiresource.PageInfo{})
}

func AddressComponentsPresenter(c *pb.AddressComponentsInfo) *apiresource.AddressComponents {
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

func AddressDetailsPresenter(resp *pb.GetAddressDetailsResponse) *apiresource.AddressDetailsResult {
	if resp == nil {
		return nil
	}
	return &apiresource.AddressDetailsResult{
		Object:           constants.ObjectTypeAddressDetailsResult,
		Address:          AddressComponentsPresenter(resp.Address),
		FormattedAddress: resp.FormattedAddress,
	}
}

func ValidatedAddressPresenter(resp *pb.ValidateAddressResponse) *apiresource.ValidatedAddress {
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
		Components:         AddressComponentsPresenter(resp.Components),
		ValidationMessages: validationMessages,
	}
}

func addressValidationStatus(isValid bool) constants.AddressValidationStatus {
	if isValid {
		return constants.AddressValidationStatusValid
	}
	return constants.AddressValidationStatusInvalid
}
