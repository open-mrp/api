package addressep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get an address.
type RetrieveAddressRequest struct {
	// Address ID.
	AddressID string `path:"id" validate:"required"`
}

type RetrieveAddressEndpoint struct{}

func (e *RetrieveAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAddressRequest, *apiresource.Address] {
	return &apiendpoint.APIEndpoint[*RetrieveAddressRequest, *apiresource.Address]{
		Title:             "Retrieve Address",
		Description:       "Retrieves an address by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/addresses/{id}",
		ContentType:       "application/json",
		Request:           &RetrieveAddressRequest{},
		Response:          &apiresource.Address{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAddressRequest) (*apiresource.Address, *apierror.APIError) {
			return svc.(AddressSvc).GetAddress
		},
	}
}
