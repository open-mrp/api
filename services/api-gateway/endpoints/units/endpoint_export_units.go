package unitep

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

// Filters which units land in the exported file.
type ExportUnitsRequest struct {
	// Free-text search term matched against unit names.
	Query *string `json:"q"`
}

var sampleExportUnitsRequest = &ExportUnitsRequest{}

func (*ExportUnitsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportUnitsRequest)
}

// Starts an export of every matching unit and returns the job that tracks it; system units
// are included, as on the list.
type ExportUnitsEndpoint struct{}

func (e *ExportUnitsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportUnitsRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*ExportUnitsRequest, *apiresource.AsyncJob]{
		Title:             "Export Units",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/units/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainUnits, Action: types.ActionRead}}},
		Preview:           true,
		ObjectType:        constants.ObjectTypeAsyncJob,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportUnitsRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(UnitSvc).ExportUnits
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}
