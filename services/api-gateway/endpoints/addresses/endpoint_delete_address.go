package addressep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to delete an address.
type DeleteAddressRequest struct {
	// Address ID.
	AddressID string `path:"id" validate:"required"`
}

// Deletes an address.
//
// Only while no order, invoice or shipment uses it and it isn't a customer's or supplier's default.
type DeleteAddressEndpoint struct{}

func (e *DeleteAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAddressRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteAddressRequest, *object.Deleted]{
		Title:             "Delete Address",
		Method:            http.MethodDelete,
		Route:             addressRoute,
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeAddress,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAddresses, Action: types.ActionDelete}}, Counterparty: authz.Counterparties(types.ActionUpdate)},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAddressRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(AddressSvc).DeleteAddress, constants.ObjectTypeAddress, "id")
		},
	})
}
