package itemcategoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Filters which item categories land in the exported file.
type ExportItemCategoriesRequest struct {
	// Free-text search term matched against category names.
	Query *string `json:"q"`
}

var sampleExportItemCategoriesRequest = &ExportItemCategoriesRequest{}

func (*ExportItemCategoriesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportItemCategoriesRequest)
}

// Starts an export of every matching item category and returns the job that tracks it.
type ExportItemCategoriesEndpoint struct{}

func (e *ExportItemCategoriesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportItemCategoriesRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*ExportItemCategoriesRequest, *apiresource.Job]{
		Title:             "Export Item Categories",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/item-categories/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportItemCategoriesRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(ItemCategorySvc).ExportItemCategories
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
