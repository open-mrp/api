package accountpriceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// CreateAccountPriceRequest is the request to create a new account price.
type CreateAccountPriceRequest struct {
	// The ID of the recipient customer account.
	RecipientAccountID string `json:"recipient_account_id" validate:"required"`
	// The ID of the product line this price applies to.
	ProductLineID string `json:"product_line_id" validate:"required"`
	// The rate value as a decimal string.
	RateValue string `json:"rate_value" validate:"required"`
	// The ID of the numerator unit for the rate.
	RateNumeratorUnitID string `json:"rate_numerator_unit_id" validate:"required"`
	// The ID of the denominator unit for the rate.
	RateDenominatorUnitID string `json:"rate_denominator_unit_id" validate:"required"`
	// The IDs of item categories to constrain this price to.
	CategoryIDs []string `json:"category_ids"`
	// The IDs of attributes to constrain this price to.
	AttributeIDs []string `json:"attribute_ids"`
}

var sampleCreateAccountPriceRequest = &CreateAccountPriceRequest{
	RecipientAccountID:    apiresource.SampleAccountID,
	ProductLineID:         apiresource.SampleProductLineID,
	RateValue:             apiresource.SampleRateValue,
	RateNumeratorUnitID:   apiresource.SampleUnitID,
	RateDenominatorUnitID: apiresource.SampleUnitID,
	CategoryIDs:           []string{apiresource.SampleItemCategoryID},
	AttributeIDs:          []string{apiresource.SampleAttributeID},
}

func (*CreateAccountPriceRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAccountPriceRequest)
}

type CreateAccountPriceEndpoint struct{}

func (e *CreateAccountPriceEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAccountPriceRequest, *apiresource.AccountPrice] {
	return &apiendpoint.APIEndpoint[*CreateAccountPriceRequest, *apiresource.AccountPrice]{
		Title:             "Create Account Price",
		Description:       "Creates a new account price for a recipient customer account. Account prices override all other pricing rules.",
		Method:            http.MethodPost,
		Route:             "/v1/sales/account-prices",
		Request:           &CreateAccountPriceRequest{},
		Response:          &apiresource.AccountPrice{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError) {
			return svc.(AccountPriceSvc).CreateAccountPrice
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountPrice,
			Fields:     []string{"recipient_account", "product_line", "categories", "attributes"},
		}),
	}
}
