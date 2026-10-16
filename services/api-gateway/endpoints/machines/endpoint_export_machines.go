package machineep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
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

func (e *ExportMachinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportMachinesRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*ExportMachinesRequest, *apiresource.AsyncJob]{
		Title:             "Export Machines",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMachines, Action: types.ActionRead}}},
		Preview:           true,
		ObjectType:        constants.ObjectTypeAsyncJob,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportMachinesRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(MachineSvc).ExportMachines
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}
