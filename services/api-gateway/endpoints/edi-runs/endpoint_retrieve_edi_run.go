package edirunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve an EDI run.
type RetrieveEDIRunRequest struct {
	// EDI run ID.
	EDIRunID string `path:"id" validate:"required"`
}

// Returns an EDI run by ID.
type RetrieveEDIRunEndpoint struct{}

func (e *RetrieveEDIRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveEDIRunRequest, *apiresource.EDIRun] {
	return (&apiendpoint.APIEndpoint[*RetrieveEDIRunRequest, *apiresource.EDIRun]{
		Title:             "Retrieve EDI Run",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/edi-runs/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeEDIRun,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveEDIRunRequest) (*apiresource.EDIRun, *apierror.APIError) {
			return svc.(EDIRunSvc).GetEDIRun
		},
	})
}
