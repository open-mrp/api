package productionrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to get a production run by ID.
type RetrieveProductionRunRequest struct {
	// Production run ID.
	ProductionRunID string `path:"id" validate:"required"`
	// Fields to include in the response.
	Includes []string `query:"include"`
}

// Returns a production run by ID.
type RetrieveProductionRunEndpoint struct{}

func (e *RetrieveProductionRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveProductionRunRequest, *apiresource.ProductionRun] {
	return (&apiendpoint.APIEndpoint[*RetrieveProductionRunRequest, *apiresource.ProductionRun]{
		Title:             "Retrieve Production Run",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-runs/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionRun,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionRuns, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveProductionRunRequest) (*apiresource.ProductionRun, *apierror.APIError) {
			return svc.(ProductionRunSvc).GetProductionRun
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductionRun,
			Fields:     []string{"responsible_user", "responsible_user.user"},
		}),
	})
}
