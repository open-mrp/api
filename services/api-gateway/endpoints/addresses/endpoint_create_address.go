package addressep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Creates an address.
//
// The address is saved to the account you are acting in, which may be your own account or a customer or supplier account you manage, and can then be used as a billing or shipping address on sales orders, invoices, and shipments.
type CreateAddressEndpoint struct{}

func (e *CreateAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*apirequest.AddressInput, *apiresource.Address] {
	return (&apiendpoint.APIEndpoint[*apirequest.AddressInput, *apiresource.Address]{
		Title:               "Create Address",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/sales/addresses",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAddresses, Action: types.ActionCreate}, {Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeAddress,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apirequest.AddressInput) (*apiresource.Address, *apierror.APIError) {
			return svc.(AddressSvc).CreateAddress
		},
		LocationFunc: func(resp *apiresource.Address) string {
			return "/v1/sales/addresses/" + resp.ID
		},
	})
}
