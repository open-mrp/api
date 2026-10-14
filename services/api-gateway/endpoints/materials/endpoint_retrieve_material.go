package materialep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to get a material.
type RetrieveMaterialRequest struct {
	// ID of the material to retrieve.
	ItemID string `path:"id" validate:"required"`
}

// Returns a material by ID.
type RetrieveMaterialEndpoint struct{}

func (e *RetrieveMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveMaterialRequest, *apiresource.Material] {
	return (&apiendpoint.APIEndpoint[*RetrieveMaterialRequest, *apiresource.Material]{
		Title:             "Retrieve Material",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/materials/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMaterials, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		Preview:           true,
		ObjectType:        constants.ObjectTypeMaterial,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveMaterialRequest) (*apiresource.Material, *apierror.APIError) {
			return svc.(MaterialSvc).GetMaterial
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMaterial,
			Fields:     []string{"item", "item.category", "item.category.properties", "item.category.unit_group", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
