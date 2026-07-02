package productlineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get a product line.
type RetrieveProductLineRequest struct {
	// Product line ID.
	ProductLineID string `path:"id" validate:"required"`
}

// Returns a product line by ID, including system-owned product lines accessible to the account.
type RetrieveProductLineEndpoint struct{}

func (e *RetrieveProductLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveProductLineRequest, *apiresource.ProductLine] {
	return (&apiendpoint.APIEndpoint[*RetrieveProductLineRequest, *apiresource.ProductLine]{
		Title:             "Retrieve Product Line",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/product-lines/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		// Reads gate on the relation helper (checkProductLineReadPermission), which
		// requires product_lines:read on the own account but customers:read /
		// suppliers:read when an internal actor reads a customer's or supplier's
		// product lines. Declare the full OR-set so the gateway gate doesn't
		// false-reject those relation-scoped reads.
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductLines, Action: types.ActionRead},
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		},
		Preview:    true,
		ObjectType: constants.ObjectTypeProductLine,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveProductLineRequest) (*apiresource.ProductLine, *apierror.APIError) {
			return svc.(ProductLineSvc).GetProductLine
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductLine,
			Fields:     []string{"owner", "owner.account", "unit_group"},
		}),
	})
}
