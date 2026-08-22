package addressep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to get an address.
type RetrieveAddressRequest struct {
	// Address ID.
	AddressID string `path:"id" validate:"required"`
}

// Retrieves an address by ID.
type RetrieveAddressEndpoint struct{}

func (e *RetrieveAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAddressRequest, *apiresource.Address] {
	return (&apiendpoint.APIEndpoint[*RetrieveAddressRequest, *apiresource.Address]{
		Title:               "Retrieve Address",
		Method:              http.MethodGet,
		Route:               "/v1/sales/addresses/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAddresses, Action: types.ActionRead}, {Domain: types.PermissionDomainCustomers, Action: types.ActionRead}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeAddress,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAddressRequest) (*apiresource.Address, *apierror.APIError) {
			return svc.(AddressSvc).GetAddress
		},
	})
}
