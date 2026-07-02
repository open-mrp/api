package accountpriceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
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
	// ID of the unit for the rate's numerator, typically a currency unit.
	RateNumeratorUnitID string `json:"rate_numerator_unit_id" validate:"required"`
	// ID of the unit for the rate's denominator — the quantity unit being priced.
	RateDenominatorUnitID string `json:"rate_denominator_unit_id" validate:"required"`
	// Item category IDs to constrain this price to.
	//
	// When empty, the price is not restricted by item category.
	CategoryIDs []string `json:"category_ids,omitzero"`
	// Attribute IDs to constrain this price to.
	//
	// When set, the price applies only to items that have every listed attribute.
	AttributeIDs []string `json:"attribute_ids,omitzero"`
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

// Creates a customer-specific price for a product line.
//
// When an order line matches the price's product line and constraints, the account price overrides standard pricing for the recipient customer.
type CreateAccountPriceEndpoint struct{}

func (e *CreateAccountPriceEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAccountPriceRequest, *apiresource.AccountPrice] {
	return (&apiendpoint.APIEndpoint[*CreateAccountPriceRequest, *apiresource.AccountPrice]{
		Title:             "Create Account Price",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-prices",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountPrice,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionCreate},
		},
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
	})
}
