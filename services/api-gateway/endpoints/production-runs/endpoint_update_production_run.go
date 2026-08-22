package productionrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to update a production run.
type UpdateProductionRunRequest struct {
	// Production run ID.
	ProductionRunID string `path:"id" validate:"required"`
	// New production run number.
	//
	// Must be unique within the account; reusing another run's number returns a conflict error.
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
// Fields not provided retain their current values. A run that has already completed can no longer be updated.
type UpdateProductionRunEndpoint struct{}

func (e *UpdateProductionRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductionRunRequest, *apiresource.ProductionRun] {
	return (&apiendpoint.APIEndpoint[*UpdateProductionRunRequest, *apiresource.ProductionRun]{
		Title:             "Update Production Run",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-runs/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionRun,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionRuns, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductionRunRequest) (*apiresource.ProductionRun, *apierror.APIError) {
			return svc.(ProductionRunSvc).UpdateProductionRun
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductionRun,
			Fields:     []string{"responsible_user", "responsible_user.user"},
		}),
	})
}
