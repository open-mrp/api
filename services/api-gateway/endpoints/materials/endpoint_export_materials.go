package materialep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Filters which materials land in the exported file.
type ExportMaterialsRequest struct {
	// Free-text search term matched against material SKU and description.
	Query *string `json:"q"`
	// Filter to materials in any of these categories.
	CategoryIDs []string `json:"category_ids"`
	// Filter to materials carrying any of these attributes.
	AttributeIDs []string `json:"attribute_ids"`
	// Filter to materials created on or after this date.
	StartDate *time.Time `json:"starts_at"`
	// Filter to materials created on or before this date.
	EndDate *time.Time `json:"ends_at"`
}

var sampleExportMaterialsRequest = &ExportMaterialsRequest{}

func (*ExportMaterialsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportMaterialsRequest)
}

// Starts an export of every matching material and returns the job that tracks it.
type ExportMaterialsEndpoint struct{}

func (e *ExportMaterialsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportMaterialsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*ExportMaterialsRequest, *apiresource.Job]{
		Title:             "Export Materials",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/materials/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportMaterialsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(MaterialSvc).ExportMaterials
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
