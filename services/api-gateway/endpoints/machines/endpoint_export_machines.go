package machineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Filters which machines land in the exported file.
type ExportMachinesRequest struct {
	// Free-text search term matched against machine names.
	Query *string `json:"q"`
}

var sampleExportMachinesRequest = &ExportMachinesRequest{}

func (*ExportMachinesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportMachinesRequest)
}

// Starts an export of every matching machine and returns the job that tracks it.
type ExportMachinesEndpoint struct{}

func (e *ExportMachinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportMachinesRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*ExportMachinesRequest, *apiresource.Job]{
		Title:             "Export Machines",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportMachinesRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(MachineSvc).ExportMachines
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
