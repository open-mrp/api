package productionrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateProductionRunRequest is the request to update an existing production run.
type UpdateProductionRunRequest struct {
	// The ID of the production run to update.
	ProductionRunID string `path:"id" validate:"required"`
	// The new production run number.
	Number *string `json:"number" validate:"omitempty,max=255"`
	// The user ID of the new responsible user.
	ResponsibleUserID *string `json:"responsible_user_id" nullable:"true" validate:"omitempty,max=191"`
}

var sampleUpdateProductionRunNumber = "PR-00042"
var sampleUpdateProductionRunUserID = apiresource.SampleUserID
var sampleUpdateProductionRunRequest = &UpdateProductionRunRequest{
	Number:            &sampleUpdateProductionRunNumber,
	ResponsibleUserID: &sampleUpdateProductionRunUserID,
}

func (*UpdateProductionRunRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateProductionRunRequest)
}

type UpdateProductionRunEndpoint struct{}

func (e *UpdateProductionRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductionRunRequest, *apiresource.ProductionRunDetail] {
	return &apiendpoint.APIEndpoint[*UpdateProductionRunRequest, *apiresource.ProductionRunDetail]{
		Title:             "Update Production Run",
		Description:       "Partially updates a production run. Only non-completed runs can be updated.",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/production-runs/{id}",
		Request:           &UpdateProductionRunRequest{},
		Response:          &apiresource.ProductionRunDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductionRunRequest) (*apiresource.ProductionRunDetail, *apierror.APIError) {
			return svc.(ProductionRunSvc).UpdateProductionRun
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductionRun,
			Fields:     []string{"responsible_user"},
		}),
	}
}
