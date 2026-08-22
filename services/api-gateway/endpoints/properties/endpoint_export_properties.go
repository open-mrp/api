package propertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Filters which properties land in the exported file.
type ExportPropertiesRequest struct {
	// Free-text search term matched against property names.
	Query *string `json:"q"`
}

var sampleExportPropertiesRequest = &ExportPropertiesRequest{}

func (*ExportPropertiesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportPropertiesRequest)
}

// Starts an export of every matching property, one row per attribute, and returns the job that tracks it.
type ExportPropertiesEndpoint struct{}

func (e *ExportPropertiesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportPropertiesRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*ExportPropertiesRequest, *apiresource.Job]{
		Title:             "Export Properties",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/properties/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportPropertiesRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(PropertySvc).ExportProperties
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
