package addressep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListAddressesRequest is the request to list addresses.
type ListAddressesRequest struct {
	apiresource.PaginationRequest
}

type ListAddressesEndpoint struct{}

func (e *ListAddressesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAddressesRequest, *apiresource.List[apiresource.Address]] {
	return &apiendpoint.APIEndpoint[*ListAddressesRequest, *apiresource.List[apiresource.Address]]{
		Title:             "List Addresses",
		Description:       "Returns a paginated list of addresses.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/addresses",
		Request:           &ListAddressesRequest{},
		Response:          &apiresource.List[apiresource.Address]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAddressesRequest) (*apiresource.List[apiresource.Address], *apierror.APIError) {
			return svc.(AddressSvc).ListAddresses
		},
	}
}
