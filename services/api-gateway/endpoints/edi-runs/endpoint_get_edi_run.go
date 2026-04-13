package edirunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve an EDI run.
type GetEDIRunRequest struct {
	// EDI run ID.
	EDIRunID string `path:"id" validate:"required"`
}

type GetEDIRunEndpoint struct{}

func (e *GetEDIRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetEDIRunRequest, *apiresource.EDIRun] {
	return &apiendpoint.APIEndpoint[*GetEDIRunRequest, *apiresource.EDIRun]{
		Title:             "Get EDI Run",
		Description:       "Returns an EDI run by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/edi-runs/{id}",
		Request:           &GetEDIRunRequest{},
		Response:          &apiresource.EDIRun{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetEDIRunRequest) (*apiresource.EDIRun, *apierror.APIError) {
			return svc.(EDIRunSvc).GetEDIRun
		},
	}
}
