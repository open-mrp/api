package addressep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Creates an address for the targeted account.
type CreateAddressEndpoint struct{}

func (e *CreateAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*apirequest.AddressInput, *apiresource.Address] {
	return (&apiendpoint.APIEndpoint[*apirequest.AddressInput, *apiresource.Address]{
		Title:             "Create Address",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/addresses",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apirequest.AddressInput) (*apiresource.Address, *apierror.APIError) {
			return svc.(AddressSvc).CreateAddress
		},
		LocationFunc: func(resp *apiresource.Address) string {
			return "/v1/sales/addresses/" + resp.ID
		},
	})
}
