package accountpriceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create an account price.
type CreateAccountPriceRequest struct {
	// ID of the customer this price is offered to.
	//
	// A price recorded against a parent customer account also applies to orders placed by its child accounts.
	RecipientAccountID string `json:"recipient_account_id" validate:"required"`
	// ID of the product line whose products this price applies to.
	ProductLineID string `json:"product_line_id" validate:"required"`
	// The price the recipient pays.
	//
	// The numerator unit is typically a currency and the denominator is the quantity unit being priced, so a matching order line takes both its unit price and its price units from this rate exactly as entered.
	Rate apirequest.RateInput `json:"rate" validate:"required"`
	// Item category IDs to record on this price.
	//
	// Order pricing matches an account price on its product line and attributes only, so categories recorded here do not narrow which products the price applies to.
	CategoryIDs []string `json:"category_ids,omitzero"`
	// Attribute IDs to constrain this price to.
	//
	// When set, the price applies only to items that have every listed attribute.
	AttributeIDs []string `json:"attribute_ids,omitzero"`
}

var sampleCreateAccountPriceRequest = &CreateAccountPriceRequest{
	RecipientAccountID: apiresource.SampleAccountID,
	ProductLineID:      apiresource.SampleProductLineID,
	Rate: apirequest.RateInput{
		Value:             apiresource.SampleRateValue,
		NumeratorUnitID:   apiresource.SampleUnitID,
		DenominatorUnitID: apiresource.SampleUnitID,
	},
	CategoryIDs:  []string{apiresource.SampleItemCategoryID},
	AttributeIDs: []string{apiresource.SampleAttributeID},
}

func (*CreateAccountPriceRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAccountPriceRequest)
}

// Creates a customer-specific price for a product line.
//
// When a sales order line for the recipient matches the price's product line and attributes, this price replaces the unit price the line would otherwise be given, including the effect of any volume discount. If more than one account price matches a line, the most recently created one wins.
type CreateAccountPriceEndpoint struct{}

func (e *CreateAccountPriceEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAccountPriceRequest, *apiresource.AccountPrice] {
	return (&apiendpoint.APIEndpoint[*CreateAccountPriceRequest, *apiresource.AccountPrice]{
		Title:             "Create Account Price",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-prices",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
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
