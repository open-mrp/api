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
	// Case-insensitive substring match on `name`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return lines with one of these statuses. Omit to return all statuses.
	Statuses []constants.ArchiveStatus `query:"statuses"`
}

// Returns a list of product lines.
//
// Includes your account's lines and the system lines every account shares. Lines are sorted by creation date, newest first.
type ListProductLinesEndpoint struct{}

func (e *ListProductLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductLinesRequest, *apiresource.List[apiresource.ProductLine]] {
	return (&apiendpoint.APIEndpoint[*ListProductLinesRequest, *apiresource.List[apiresource.ProductLine]]{
		Title:             "List Product Lines",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             productLinesRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProductLines, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ObjectType:        constants.ObjectTypeProductLine,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductLinesRequest) (*apiresource.List[apiresource.ProductLine], *apierror.APIError) {
			return svc.(ProductLineSvc).ListProductLines
		},
		IncludeConfig: productLineIncludes(),
	})
}
