package accountpriceep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to delete an account price.
type DeleteAccountPriceRequest struct {
	// Account price ID.
	AccountPriceID string `path:"id" validate:"required"`
}

// Deletes an account price.
//
// The price's category and attribute associations and its rate record are removed with it. Deletion is permanent; further requests against the deleted ID return an error.
//
// Order lines that have already been priced keep the unit price they were given; only lines priced after the deletion revert to standard pricing.
type DeleteAccountPriceEndpoint struct{}

func (e *DeleteAccountPriceEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAccountPriceRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteAccountPriceRequest, *object.Deleted]{
		Title:             "Delete Account Price",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/account-prices/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeAccountPrice,
		Public:            false,
		AgentTool:         true,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionDelete},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAccountPriceRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(AccountPriceSvc).DeleteAccountPrice, constants.ObjectTypeAccountPrice, "id")
		},
	})
}
