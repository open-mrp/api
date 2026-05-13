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

// Request to create an account price.
type CreateAccountPriceRequest struct {
	// Recipient customer account ID.
	RecipientAccountID string `json:"recipient_account_id" validate:"required"`
	// Product line ID.
	ProductLineID string `json:"product_line_id" validate:"required"`
	// Rate value as a decimal string.
	RateValue string `json:"rate_value" validate:"required"`
	// Rate numerator unit ID.
	RateNumeratorUnitID string `json:"rate_numerator_unit_id" validate:"required"`
	// Rate denominator unit ID.
	RateDenominatorUnitID string `json:"rate_denominator_unit_id" validate:"required"`
	// Item category IDs to constrain this price to.
	CategoryIDs []string `json:"category_ids"`
	// Attribute IDs to constrain this price to.
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
		Description:       "Creates an account price for a recipient customer account. Account prices override all other pricing rules.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-prices",
		Request:           &CreateAccountPriceRequest{},
		Response:          &apiresource.AccountPrice{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError) {
			return svc.(AccountPriceSvc).CreateAccountPrice
		},
		LocationFunc: func(resp *apiresource.AccountPrice) string {
			return "/v1/sales/account-prices/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountPrice,
			Fields:     []string{"recipient_account", "product_line", "categories", "attributes"},
		}),
	}
}
