package productlineep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list product lines.
type ListProductLinesRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// Returns a paginated list of product lines, newest first.
//
// Covers both the product lines your account owns and the shared system lines. The `q` search term is matched against the product line name.
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
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProductLines, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductLine,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductLinesRequest) (*apiresource.List[apiresource.ProductLine], *apierror.APIError) {
			return svc.(ProductLineSvc).ListProductLines
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductLine,
			Fields:     []string{"owner", "owner.account", "unit_group", "default_lot"},
		}),
	})
}
