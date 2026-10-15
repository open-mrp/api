package productlineep

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

// Request to retrieve a product line.
type RetrieveProductLineRequest struct {
	// Product line ID.
	ProductLineID string `path:"id" validate:"required"`
}

// Retrieves a product line.
//
// Works for your account's lines and for system lines.
type RetrieveProductLineEndpoint struct{}

func (e *RetrieveProductLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveProductLineRequest, *apiresource.ProductLine] {
	return (&apiendpoint.APIEndpoint[*RetrieveProductLineRequest, *apiresource.ProductLine]{
		Title:             "Retrieve Product Line",
		Method:            http.MethodGet,
		Route:             productLineRoute,
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProductLines, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ObjectType:        constants.ObjectTypeProductLine,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveProductLineRequest) (*apiresource.ProductLine, *apierror.APIError) {
			return svc.(ProductLineSvc).GetProductLine
		},
		IncludeConfig: productLineIncludes(),
	})
}
