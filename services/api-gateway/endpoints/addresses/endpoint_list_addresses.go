package addressep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list addresses.
type ListAddressesRequest struct {
	apiresource.PaginationRequest
	// Filter by address type.
	Type *constants.AddressType `query:"type"`
}

// Returns a paginated list of addresses.
type ListAddressesEndpoint struct{}

func (e *ListAddressesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAddressesRequest, *apiresource.List[apiresource.Address]] {
	return (&apiendpoint.APIEndpoint[*ListAddressesRequest, *apiresource.List[apiresource.Address]]{
		Title:             "List Addresses",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/addresses",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAddress,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAddressesRequest) (*apiresource.List[apiresource.Address], *apierror.APIError) {
			return svc.(AddressSvc).ListAddresses
		},
	})
}
