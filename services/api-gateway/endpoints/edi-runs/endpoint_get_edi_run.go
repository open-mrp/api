package edirunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetEDIRunRequest is the request to retrieve a single EDI run.
type GetEDIRunRequest struct {
	// The ID of the EDI run to retrieve.
	EDIRunID string `path:"id" validate:"required"`
}

type GetEDIRunEndpoint struct{}

func (e *GetEDIRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetEDIRunRequest, *apiresource.EDIRun] {
	return &apiendpoint.APIEndpoint[*GetEDIRunRequest, *apiresource.EDIRun]{
		Title:             "Get EDI Run",
		Description:       "Returns a single EDI run by its ID.",
		Method:            http.MethodGet,
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
