package addressep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
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
		Title:             "Delete Address",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/addresses/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAddresses, Action: types.ActionDelete}}, Counterparty: authz.Counterparties(types.ActionUpdate)},
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAddressRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AddressSvc).DeleteAddress
		},
	})
}
