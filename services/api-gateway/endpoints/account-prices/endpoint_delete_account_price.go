package accountpriceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
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

func (e *DeleteAccountPriceEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAccountPriceRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteAccountPriceRequest, *apiresource.EmptyResource]{
		Title:             "Delete Account Price",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/account-prices/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionDelete},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAccountPriceRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountPriceSvc).DeleteAccountPrice
		},
	})
}
