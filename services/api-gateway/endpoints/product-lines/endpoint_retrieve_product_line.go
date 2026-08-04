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

// Returns a single product line by ID.
//
// Both the product lines your account owns and the shared system lines can be retrieved.
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
			Fields:     []string{"owner", "owner.account", "unit_group", "default_lot", "default_lot.unit"},
		}),
	})
}
