package addressep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteAddressRequest is the request to delete an address.
type DeleteAddressRequest struct {
	// The ID of the address to delete.
	AddressID string `path:"id" validate:"required"`
}

type DeleteAddressEndpoint struct{}

func (e *DeleteAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAddressRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteAddressRequest, *apiresource.EmptyResource]{
		Title:             "Delete Address",
		Description:       "Deletes an address. This request will fail if the address is in use as a billing or shipping address on a sales order, invoice, or shipment, or as a default address on the account.",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/addresses/{id}",
		ContentType:       "application/json",
		Request:           &DeleteAddressRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAddressRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AddressSvc).DeleteAddress
		},
	}
}
