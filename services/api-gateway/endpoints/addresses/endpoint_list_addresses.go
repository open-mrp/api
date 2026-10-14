package addressep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list addresses.
type ListAddressesRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Filters results to addresses of the given type.
	Type *constants.AddressType `query:"type"`
}

// Returns a paginated list of addresses.
//
// Addresses belonging to the account you are acting in are returned newest first. The `q` search term matches the address name, street lines, city, state, postal code, and country.
type ListAddressesEndpoint struct{}

func (e *ListAddressesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAddressesRequest, *apiresource.List[apiresource.Address]] {
	return (&apiendpoint.APIEndpoint[*ListAddressesRequest, *apiresource.List[apiresource.Address]]{
		Title:             "List Addresses",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/addresses",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAddresses, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		Preview:           true,
		ObjectType:        constants.ObjectTypeAddress,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAddressesRequest) (*apiresource.List[apiresource.Address], *apierror.APIError) {
			return svc.(AddressSvc).ListAddresses
		},
	})
}
