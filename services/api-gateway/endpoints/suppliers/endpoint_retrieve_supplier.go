package supplierep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to retrieve a supplier by ID.
type RetrieveSupplierRequest struct {
	// Supplier ID.
	SupplierID string `path:"id" validate:"required"`
}

// Returns a supplier by ID.
type RetrieveSupplierEndpoint struct{}

func (e *RetrieveSupplierEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveSupplierRequest, *apiresource.Supplier] {
	return (&apiendpoint.APIEndpoint[*RetrieveSupplierRequest, *apiresource.Supplier]{
		Title:             "Retrieve Supplier",
		Method:            http.MethodGet,
		Route:             "/v1/operations/suppliers/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSupplier,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveSupplierRequest) (*apiresource.Supplier, *apierror.APIError) {
			return svc.(SupplierSvc).GetSupplier
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSupplier,
			Fields:     []string{"bill_to_address", "ship_to_address"},
		}),
	})
}
