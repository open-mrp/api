package materialep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list materials.
type ListMaterialsRequest struct {
	apiresource.PaginationRequest
	// Filter by category IDs.
	CategoryIDs []string `query:"category_ids"`
	// Filter by attribute IDs.
	AttributeIDs []string `query:"attribute_ids"`
	// Filter to materials created on or after this date.
	StartDate *time.Time `query:"start_date"`
	// Filter to materials created on or before this date.
	EndDate *time.Time `query:"end_date"`
}

// Returns a paginated list of materials.
type ListMaterialsEndpoint struct{}

func (e *ListMaterialsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMaterialsRequest, *apiresource.List[apiresource.Material]] {
	return (&apiendpoint.APIEndpoint[*ListMaterialsRequest, *apiresource.List[apiresource.Material]]{
		Title:             "List Materials",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/materials",
		Request:           &ListMaterialsRequest{},
		Response:          &apiresource.List[apiresource.Material]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMaterialsRequest) (*apiresource.List[apiresource.Material], *apierror.APIError) {
			return svc.(MaterialSvc).ListMaterials
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMaterial,
			Fields:     []string{"item", "item.category", "item.category.properties", "item.category.unit_group", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	}).WithDocSource(e)
}
