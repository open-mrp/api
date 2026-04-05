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

type ListMaterialsRequest struct {
	apiresource.PaginationRequest
	CategoryIDs  []string   `query:"category_ids"`
	AttributeIDs []string   `query:"attribute_ids"`
	StartDate    *time.Time `query:"start_date"`
	EndDate      *time.Time `query:"end_date"`
}

type ListMaterialsEndpoint struct{}

func (e *ListMaterialsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMaterialsRequest, *apiresource.List[apiresource.Material]] {
	return &apiendpoint.APIEndpoint[*ListMaterialsRequest, *apiresource.List[apiresource.Material]]{
		Title:             "List Materials",
		Description:       "Returns a paginated list of materials for the current account.",
		Method:            http.MethodGet,
		Route:             "/v1/operations/materials",
		Request:           &ListMaterialsRequest{},
		Response:          &apiresource.List[apiresource.Material]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMaterialsRequest) (*apiresource.List[apiresource.Material], *apierror.APIError) {
			return svc.(MaterialSvc).ListMaterials
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMaterial,
			Fields:     []string{"item", "item.category", "item.unit_value", "item.unit_cost", "item.burn_rate"},
		}),
	}
}
