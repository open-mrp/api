package unitep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
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

func (e *ExportUnitsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportUnitsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*ExportUnitsRequest, *apiresource.Job]{
		Title:             "Export Units",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/units/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportUnitsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(UnitSvc).ExportUnits
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
