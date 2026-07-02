package addressep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete an address.
type DeleteAddressRequest struct {
	// Address ID.
	AddressID string `path:"id" validate:"required"`
}

// Deletes an address.
//
// Deletion fails if the address is in use as a billing or shipping address on a sales order, invoice, or shipment, or as a default account address.
type DeleteAddressEndpoint struct{}

func (e *DeleteAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAddressRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteAddressRequest, *apiresource.EmptyResource]{
		Title:               "Delete Address",
		Method:              http.MethodDelete,
		Route:               "/v1/sales/addresses/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAddresses, Action: types.ActionDelete}, {Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAddressRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AddressSvc).DeleteAddress
		},
	})
}
