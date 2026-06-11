package productionrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to update a production run.
type UpdateProductionRunRequest struct {
	// Production run ID.
	ProductionRunID string `path:"id" validate:"required"`
	// New production run number.
	//
	// Must be unique within the account.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// ID of the account user accountable for executing the run.
	//
	// Accepts either an account user ID or a user ID; it is resolved and stored as the account user.
	ResponsibleUserID field.Optional[string] `json:"responsible_user_id,omitzero" validate:"omitempty"`
}

var sampleUpdateProductionRunNumber = "PR-00042"
var sampleUpdateProductionRunUserID = apiresource.SampleUserID
var sampleUpdateProductionRunRequest = &UpdateProductionRunRequest{
	Number:            field.Some(sampleUpdateProductionRunNumber),
	ResponsibleUserID: field.Some(sampleUpdateProductionRunUserID),
}

func (*UpdateProductionRunRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateProductionRunRequest)
}

// Partially updates a production run.
//
// Fails if the run has been completed.
type UpdateProductionRunEndpoint struct{}

func (e *UpdateProductionRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductionRunRequest, *apiresource.ProductionRunDetail] {
	return (&apiendpoint.APIEndpoint[*UpdateProductionRunRequest, *apiresource.ProductionRunDetail]{
		Title:             "Update Production Run",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-runs/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionRun,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductionRunRequest) (*apiresource.ProductionRunDetail, *apierror.APIError) {
			return svc.(ProductionRunSvc).UpdateProductionRun
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductionRun,
			Fields:     []string{"responsible_user", "responsible_user.user"},
		}),
	})
}
