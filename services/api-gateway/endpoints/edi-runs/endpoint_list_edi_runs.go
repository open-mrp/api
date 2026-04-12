package edirunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListEDIRunsRequest is the request to list EDI runs with optional filters.
type ListEDIRunsRequest struct {
	apiresource.PaginationRequest
	// Filter by whether the EDI run succeeded.
	HasSucceeded *bool `query:"has_succeeded"`
}

type ListEDIRunsEndpoint struct{}

func (e *ListEDIRunsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListEDIRunsRequest, *apiresource.List[apiresource.EDIRun]] {
	return &apiendpoint.APIEndpoint[*ListEDIRunsRequest, *apiresource.List[apiresource.EDIRun]]{
		Title:             "List EDI Runs",
		Description:       "Returns a paginated list of EDI runs for the target account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/edi-runs",
		Request:           &ListEDIRunsRequest{},
		Response:          &apiresource.List[apiresource.EDIRun]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListEDIRunsRequest) (*apiresource.List[apiresource.EDIRun], *apierror.APIError) {
			return svc.(EDIRunSvc).ListEDIRuns
		},
	}
}
