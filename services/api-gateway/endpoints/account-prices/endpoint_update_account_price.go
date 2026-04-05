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

// UpdateAccountPriceRequest is the request to partially update an account price.
type UpdateAccountPriceRequest struct {
	// The ID of the account price to update.
	AccountPriceID string `path:"id" validate:"required"`
	// The ID of the recipient (customer) account.
	RecipientAccountID *string `json:"recipient_account_id,omitempty"`
	// The ID of the product line this price applies to.
	ProductLineID *string `json:"product_line_id,omitempty"`
	// The rate value as a decimal string.
	RateValue *string `json:"rate_value,omitempty"`
	// The ID of the numerator unit for the rate.
	RateNumeratorUnitID *string `json:"rate_numerator_unit_id,omitempty"`
	// The ID of the denominator unit for the rate.
	RateDenominatorUnitID *string `json:"rate_denominator_unit_id,omitempty"`
	// The IDs of item categories to constrain this price to. Replaces existing categories.
	CategoryIDs *[]string `json:"category_ids,omitempty"`
	// The IDs of attributes to constrain this price to. Replaces existing attributes.
	AttributeIDs *[]string `json:"attribute_ids,omitempty"`
}

var sampleUpdateAccountPriceRequest = &UpdateAccountPriceRequest{
	RateValue: new("30.000000000000000000000000000000"),
}

func (*UpdateAccountPriceRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountPriceRequest)
}

type UpdateAccountPriceEndpoint struct{}

func (e *UpdateAccountPriceEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountPriceRequest, *apiresource.AccountPrice] {
	return &apiendpoint.APIEndpoint[*UpdateAccountPriceRequest, *apiresource.AccountPrice]{
		Title:             "Update Account Price",
		Description:       "Partially updates an account price. If category_ids or attribute_ids are provided, they replace the existing set entirely.",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/account-prices/{id}",
		ContentType:       "application/json",
		Request:           &UpdateAccountPriceRequest{},
		Response:          &apiresource.AccountPrice{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError) {
			return svc.(AccountPriceSvc).UpdateAccountPrice
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountPrice,
			Fields:     []string{"recipient_account", "product_line", "categories", "attributes"},
		}),
	}
}
