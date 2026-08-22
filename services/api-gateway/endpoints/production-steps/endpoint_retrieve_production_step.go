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

// Request to retrieve a production step.
type RetrieveProductionStepRequest struct {
	// Production step ID.
	ProductionStepID string `path:"id" validate:"required"`
}

// Returns a production step by ID.
type RetrieveProductionStepEndpoint struct{}

func (e *RetrieveProductionStepEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveProductionStepRequest, *apiresource.ProductionStep] {
	return (&apiendpoint.APIEndpoint[*RetrieveProductionStepRequest, *apiresource.ProductionStep]{
		Title:               "Retrieve Production Step",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/operations/production-steps/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProductionSteps, Action: types.ActionRead}},
		ObjectType:          constants.ObjectTypeProductionStep,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveProductionStepRequest) (*apiresource.ProductionStep, *apierror.APIError) {
			return svc.(ProductionStepSvc).GetProductionStep
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductionStep,
			Fields: []string{
				"production",
				"production.produced_item",
				"consumptions",
				"consumptions.consumed_item",
				"consumptions.quantity",
				"consumptions.waste_quantity",
				"machines",
				"scanning_station",
				"department",
				"in_steps",
				"out_steps",
			},
		}),
	})
}
