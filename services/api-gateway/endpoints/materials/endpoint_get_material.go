package materialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

type GetMaterialRequest struct {
	ItemID string `path:"id" validate:"required"`
}

type GetMaterialEndpoint struct{}

func (e *GetMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetMaterialRequest, *apiresource.Material] {
	return &apiendpoint.APIEndpoint[*GetMaterialRequest, *apiresource.Material]{
		Title:             "Get Material",
		Description:       "Returns a single material by its ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/materials/{id}",
		Request:           &GetMaterialRequest{},
		Response:          &apiresource.Material{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetMaterialRequest) (*apiresource.Material, *apierror.APIError) {
			return svc.(MaterialSvc).GetMaterial
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMaterial,
			Fields:     []string{"item", "item.category", "item.unit_value", "item.unit_cost", "item.burn_rate"},
		}),
	}
}
