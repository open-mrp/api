package locationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Filters which storage locations land in the exported file.
type ExportLocationsRequest struct {
	// Free-text search term matched against location names.
	Query *string `json:"q"`
}

var sampleExportLocationsRequest = &ExportLocationsRequest{}

func (*ExportLocationsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportLocationsRequest)
}

// Starts an export of every matching storage location and returns the job that tracks it.
type ExportLocationsEndpoint struct{}

func (e *ExportLocationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportLocationsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*ExportLocationsRequest, *apiresource.Job]{
		Title:             "Export Storage Locations",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/locations/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportLocationsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(LocationSvc).ExportLocations
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
