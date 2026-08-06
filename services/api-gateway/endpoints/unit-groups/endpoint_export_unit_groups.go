package unitgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Filters which unit groups land in the exported file.
type ExportUnitGroupsRequest struct {
	// Free-text search term matched against unit group names.
	Query *string `json:"q"`
}

var sampleExportUnitGroupsRequest = &ExportUnitGroupsRequest{}

func (*ExportUnitGroupsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportUnitGroupsRequest)
}

// Starts an export of every matching unit group and returns the job that tracks it; the file lists
// a group's units one per row, the base unit excepted.
type ExportUnitGroupsEndpoint struct{}

func (e *ExportUnitGroupsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportUnitGroupsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*ExportUnitGroupsRequest, *apiresource.Job]{
		Title:             "Export Unit Groups",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/unit-groups/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportUnitGroupsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(UnitGroupSvc).ExportUnitGroups
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
