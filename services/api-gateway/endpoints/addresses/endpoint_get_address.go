package addressep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetAddressRequest is the request to retrieve a single address.
type GetAddressRequest struct {
	// The ID of the address to retrieve.
	AddressID string `path:"id" validate:"required"`
}

type GetAddressEndpoint struct{}

func (e *GetAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAddressRequest, *apiresource.Address] {
	return &apiendpoint.APIEndpoint[*GetAddressRequest, *apiresource.Address]{
		Title:             "Get Address",
		Description:       "Returns a single address by its ID.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/addresses/{id}",
		ContentType:       "application/json",
		Request:           &GetAddressRequest{},
		Response:          &apiresource.Address{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetAddressRequest) (*apiresource.Address, *apierror.APIError) {
			return svc.(AddressSvc).GetAddress
		},
	}
}
