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

// CreateProductionRunRequest is the request to create a new production run.
type CreateProductionRunRequest struct {
	// The user ID of the user responsible for this production run.
	ResponsibleUserID string `json:"responsible_user_id" validate:"required,max=191"`
}

var sampleCreateProductionRunRequest = &CreateProductionRunRequest{
	ResponsibleUserID: apiresource.SampleUserID,
}

func (*CreateProductionRunRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateProductionRunRequest)
}

type CreateProductionRunEndpoint struct{}

func (e *CreateProductionRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductionRunRequest, *apiresource.ProductionRunDetail] {
	return &apiendpoint.APIEndpoint[*CreateProductionRunRequest, *apiresource.ProductionRunDetail]{
		Title:             "Create Production Run",
		Description:       "Creates a new production run for the current account.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-runs",
		Request:           &CreateProductionRunRequest{},
		Response:          &apiresource.ProductionRunDetail{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateProductionRunRequest) (*apiresource.ProductionRunDetail, *apierror.APIError) {
			return svc.(ProductionRunSvc).CreateProductionRun
		},
		LocationFunc: func(resp *apiresource.ProductionRunDetail) string {
			return "/v1/operations/production-runs/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductionRun,
			Fields:     []string{"responsible_user"},
		}),
	}
}
