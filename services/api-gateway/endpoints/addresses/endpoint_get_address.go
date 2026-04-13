package addressep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get an address.
type GetAddressRequest struct {
	// Address ID.
	AddressID string `path:"id" validate:"required"`
}

type GetAddressEndpoint struct{}

func (e *GetAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAddressRequest, *apiresource.Address] {
	return &apiendpoint.APIEndpoint[*GetAddressRequest, *apiresource.Address]{
		Title:             "Get Address",
		Description:       "Returns an address by ID.",
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
