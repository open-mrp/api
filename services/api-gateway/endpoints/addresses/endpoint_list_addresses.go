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
	// Case-insensitive substring match on `name`, `line_1`, `line_2`, `city`, `state`, `postal_code` or `country`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return addresses of these types.
	Types []constants.AddressType `query:"types"`
}

// Returns a list of addresses.
//
// Returns the addresses of the account you're acting in, newest first.
type ListAddressesEndpoint struct{}

func (e *ListAddressesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAddressesRequest, *apiresource.List[apiresource.Address]] {
	return (&apiendpoint.APIEndpoint[*ListAddressesRequest, *apiresource.List[apiresource.Address]]{
		Title:             "List Addresses",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             addressesRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAddresses, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ObjectType:        constants.ObjectTypeAddress,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAddressesRequest) (*apiresource.List[apiresource.Address], *apierror.APIError) {
			return svc.(AddressSvc).ListAddresses
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAddress,
			Fields:     []string{"receive_calendar"},
		}),
	})
}
