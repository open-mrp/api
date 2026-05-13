package addressep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

type CreateAddressEndpoint struct{}

func (e *CreateAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*apirequest.AddressInput, *apiresource.Address] {
	return &apiendpoint.APIEndpoint[*apirequest.AddressInput, *apiresource.Address]{
		Title:             "Create Address",
		Description:       "Creates an address for the targeted account.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/addresses",
		Request:           &apirequest.AddressInput{},
		Response:          &apiresource.Address{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apirequest.AddressInput) (*apiresource.Address, *apierror.APIError) {
			return svc.(AddressSvc).CreateAddress
		},
		LocationFunc: func(resp *apiresource.Address) string {
			return "/v1/sales/addresses/" + resp.ID
		},
	}
}
