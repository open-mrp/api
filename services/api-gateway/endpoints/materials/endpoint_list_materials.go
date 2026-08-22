package materialep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list materials.
type ListMaterialsRequest struct {
	apiresource.PaginationRequest
	// Filter to materials in any of these categories.
	CategoryIDs []string `query:"category_ids"`
	// Filter to materials carrying any of these attributes.
	AttributeIDs []string `query:"attribute_ids"`
	// Filter to materials created on or after this date.
	StartDate *time.Time `query:"starts_at"`
	// Filter to materials created on or before this date.
	EndDate *time.Time `query:"ends_at"`
}

// Returns a paginated list of materials, newest first.
//
// `q` matches against SKU and description, with closer SKU matches ranked first.
type ListMaterialsEndpoint struct{}

func (e *ListMaterialsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMaterialsRequest, *apiresource.List[apiresource.Material]] {
	return (&apiendpoint.APIEndpoint[*ListMaterialsRequest, *apiresource.List[apiresource.Material]]{
		Title:               "List Materials",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/catalog/materials",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMaterials, Action: types.ActionRead}, {Domain: types.PermissionDomainCustomers, Action: types.ActionRead}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeMaterial,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMaterialsRequest) (*apiresource.List[apiresource.Material], *apierror.APIError) {
			return svc.(MaterialSvc).ListMaterials
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMaterial,
			Fields:     []string{"item", "item.category", "item.category.properties", "item.category.unit_group", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
