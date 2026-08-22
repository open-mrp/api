package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve a production output.
type RetrieveProductionRequest struct {
	// Production step ID.
	ProductionStepID string `path:"production_step_id" validate:"required"`
	// Production ID.
	ProductionID string `path:"id" validate:"required"`
}

// Returns a production output by ID within a production step.
type RetrieveProductionEndpoint struct{}

func (e *RetrieveProductionEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveProductionRequest, *apiresource.ProductionOutput] {
	return (&apiendpoint.APIEndpoint[*RetrieveProductionRequest, *apiresource.ProductionOutput]{
		Title:               "Retrieve Production",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/operations/production-steps/{production_step_id}/productions/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProductionSteps, Action: types.ActionRead}},
		ObjectType:          constants.ObjectTypeProduction,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveProductionRequest) (*apiresource.ProductionOutput, *apierror.APIError) {
			return svc.(ProductionStepSvc).GetProduction
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProduction,
			Fields:     []string{"produced_item"},
		}),
	})
}
