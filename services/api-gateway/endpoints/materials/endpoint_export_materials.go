package materialep

import (
	"context"
	"net/http"
	"time"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
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

func (e *ExportMaterialsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportMaterialsRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*ExportMaterialsRequest, *apiresource.AsyncJob]{
		Title:             "Export Materials",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/materials/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMaterials, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ObjectType:        constants.ObjectTypeAsyncJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAsyncJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportMaterialsRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(MaterialSvc).ExportMaterials
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}
