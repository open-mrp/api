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

// Request to list product lines.
type ListProductLinesRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of product lines, including account-owned and system product lines.
type ListProductLinesEndpoint struct{}

func (e *ListProductLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductLinesRequest, *apiresource.List[apiresource.ProductLine]] {
	return (&apiendpoint.APIEndpoint[*ListProductLinesRequest, *apiresource.List[apiresource.ProductLine]]{
		Title:             "List Product Lines",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/product-lines",
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductLinesRequest) (*apiresource.List[apiresource.ProductLine], *apierror.APIError) {
			return svc.(ProductLineSvc).ListProductLines
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductLine,
			Fields:     []string{"owner", "owner.account", "unit_group", "default_lot", "default_lot.unit"},
		}),
	})
}
