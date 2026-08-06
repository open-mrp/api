package partep

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

// Filters which parts land in the exported file.
type ExportPartsRequest struct {
	// Free-text search term matched against the part's SKU or description.
	Query *string `json:"q"`
	// Filter to parts belonging to any of these item categories.
	CategoryIDs []string `json:"category_ids"`
	// Filter to parts carrying at least one of these attributes.
	AttributeIDs []string `json:"attribute_ids"`
	// Filter to parts created at or after this time.
	StartDate *time.Time `json:"starts_at"`
	// Filter to parts created at or before this time.
	EndDate *time.Time `json:"ends_at"`
}

var sampleExportPartsRequest = &ExportPartsRequest{}

func (*ExportPartsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportPartsRequest)
}

// Starts an export of every matching part and returns the job that tracks it.
type ExportPartsEndpoint struct{}

func (e *ExportPartsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportPartsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*ExportPartsRequest, *apiresource.Job]{
		Title:             "Export Parts",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/parts/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportPartsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(PartSvc).ExportParts
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
