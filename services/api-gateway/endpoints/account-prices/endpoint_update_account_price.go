package accountpriceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to partially update an account price.
type UpdateAccountPriceRequest struct {
	// Account price ID.
	AccountPriceID string `path:"id" validate:"required"`
	// Recipient account ID.
	RecipientAccountID field.Optional[string] `json:"recipient_account_id,omitzero" validate:"omitempty"`
	// Product line ID.
	ProductLineID field.Optional[string] `json:"product_line_id,omitzero" validate:"omitempty"`
	// Rate value as a decimal string.
	RateValue field.Optional[string] `json:"rate_value,omitzero"`
	// Rate numerator unit ID.
	RateNumeratorUnitID field.Optional[string] `json:"rate_numerator_unit_id,omitzero" validate:"omitempty"`
	// Rate denominator unit ID.
	RateDenominatorUnitID field.Optional[string] `json:"rate_denominator_unit_id,omitzero" validate:"omitempty"`
	// Item category IDs to constrain this price to. Replaces existing categories.
	CategoryIDs field.Optional[[]string] `json:"category_ids,omitzero"`
	// Attribute IDs to constrain this price to. Replaces existing attributes.
	AttributeIDs field.Optional[[]string] `json:"attribute_ids,omitzero"`
}

var sampleUpdateAccountPriceRequest = &UpdateAccountPriceRequest{
	RateValue: field.Some("30.000000000000000000000000000000"),
}

func (*UpdateAccountPriceRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountPriceRequest)
}

// Partially updates an account price. If category_ids or attribute_ids are provided, they replace the existing set entirely.
type UpdateAccountPriceEndpoint struct{}

func (e *UpdateAccountPriceEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountPriceRequest, *apiresource.AccountPrice] {
	return (&apiendpoint.APIEndpoint[*UpdateAccountPriceRequest, *apiresource.AccountPrice]{
		Title:             "Update Account Price",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/account-prices/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountPrice,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError) {
			return svc.(AccountPriceSvc).UpdateAccountPrice
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountPrice,
			Fields:     []string{"recipient_account", "product_line", "categories", "attributes"},
		}),
	})
}
