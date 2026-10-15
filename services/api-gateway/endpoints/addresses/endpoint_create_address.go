package addressep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Creates an address.
//
// Saved to the account you're acting in: yours, or a customer's or supplier's you manage. Use it as a billing or shipping address on orders, invoices and shipments.
type CreateAddressEndpoint struct{}

func (e *CreateAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*apirequest.AddressInput, *apiresource.Address] {
	return (&apiendpoint.APIEndpoint[*apirequest.AddressInput, *apiresource.Address]{
		Title:             "Create Address",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             addressesRoute,
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAddresses, Action: types.ActionCreate}}, Counterparty: authz.Counterparties(types.ActionUpdate)},
		ObjectType:        constants.ObjectTypeAddress,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apirequest.AddressInput) (*apiresource.Address, *apierror.APIError) {
			return svc.(AddressSvc).CreateAddress
		},
		LocationFunc: func(resp *apiresource.Address) string {
			return addressesRoute + "/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAddress,
			Fields:     []string{"receive_calendar"},
		}),
	})
}
