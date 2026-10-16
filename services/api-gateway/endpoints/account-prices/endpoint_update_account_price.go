package accountpriceep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to partially update an account price.
type UpdateAccountPriceRequest struct {
	// Account price ID.
	AccountPriceID string `path:"id" validate:"required"`
	// ID of the customer this price is offered to.
	RecipientAccountID field.Optional[string] `json:"recipient_account_id,omitzero" validate:"omitempty"`
	// ID of the product line whose products this price applies to.
	ProductLineID field.Optional[string] `json:"product_line_id,omitzero" validate:"omitempty"`
	// The price the recipient pays.
	//
	// Supplied whole: send the value together with both units, since the rate is replaced rather than merged field by field.
	Rate field.Optional[apirequest.RateInput] `json:"rate,omitzero"`
	// Item category IDs to record on this price.
	//
	// When provided, replaces the existing set of categories entirely; an empty list removes them all. Categories are recorded only — they do not narrow which products the price applies to.
	CategoryIDs field.Optional[[]string] `json:"category_ids,omitzero"`
	// Attribute IDs to constrain this price to.
	//
	// When provided, replaces the existing set of attributes entirely; an empty list removes all attribute constraints.
	AttributeIDs field.Optional[[]string] `json:"attribute_ids,omitzero"`
}

var sampleUpdateAccountPriceRequest = &UpdateAccountPriceRequest{
	Rate: field.Some(apirequest.RateInput{
		Value:     "30.000000000000000000000000000000",
		UnitID:    apiresource.SampleUnitID,
		PerUnitID: apiresource.SampleUnitID,
	}),
}

func (*UpdateAccountPriceRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountPriceRequest)
}

// Partially updates an account price.
//
// Only the provided fields are changed. If `category_ids` or `attribute_ids` are provided, they replace the existing set entirely.
//
// Order lines that have already been priced keep the unit price they were given; the new price applies to lines priced after the change.
type UpdateAccountPriceEndpoint struct{}

func (e *UpdateAccountPriceEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountPriceRequest, *apiresource.AccountPrice] {
	return (&apiendpoint.APIEndpoint[*UpdateAccountPriceRequest, *apiresource.AccountPrice]{
		Title:             "Update Account Price",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/account-prices/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeAccountPrice,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionUpdate},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError) {
			return svc.(AccountPriceSvc).UpdateAccountPrice
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountPrice,
			Fields:     []string{"recipient_account", "product_line", "categories", "attributes"},
		}),
	})
}
