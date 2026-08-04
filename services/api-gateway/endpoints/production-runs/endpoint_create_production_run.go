package productionrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a production run.
type CreateProductionRunRequest struct {
	// ID of the account user accountable for executing the run.
	//
	// Accepts either an account user ID or a user ID; it is resolved and stored as the account user.
	ResponsibleUserID string `json:"responsible_user_id" validate:"required"`
}

var sampleCreateProductionRunRequest = &CreateProductionRunRequest{
	ResponsibleUserID: apiresource.SampleUserID,
}

func (*CreateProductionRunRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateProductionRunRequest)
}

// Creates a production run.
//
// The run number is assigned automatically as the next sequential number for the account. The new run starts empty and neither started nor completed; add the work to be run with the add-batches endpoint.
type CreateProductionRunEndpoint struct{}

func (e *CreateProductionRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductionRunRequest, *apiresource.ProductionRun] {
	return (&apiendpoint.APIEndpoint[*CreateProductionRunRequest, *apiresource.ProductionRun]{
		Title:             "Create Production Run",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-runs",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionRun,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionRuns, Action: types.ActionCreate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateProductionRunRequest) (*apiresource.ProductionRun, *apierror.APIError) {
			return svc.(ProductionRunSvc).CreateProductionRun
		},
		LocationFunc: func(resp *apiresource.ProductionRun) string {
			return "/v1/operations/production-runs/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductionRun,
			Fields:     []string{"responsible_user", "responsible_user.user"},
		}),
	})
}
