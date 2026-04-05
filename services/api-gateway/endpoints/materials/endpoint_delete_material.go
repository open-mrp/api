package materialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

type DeleteMaterialRequest struct {
	ItemID string `path:"id" validate:"required"`
}

type DeleteMaterialEndpoint struct{}

func (e *DeleteMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteMaterialRequest, *apiresource.Material] {
	return &apiendpoint.APIEndpoint[*DeleteMaterialRequest, *apiresource.Material]{
		Title:             "Delete Material",
		Description:       "Deletes a material by its ID.",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/materials/{id}",
		Request:           &DeleteMaterialRequest{},
		Response:          &apiresource.Material{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteMaterialRequest) (*apiresource.Material, *apierror.APIError) {
			return svc.(MaterialSvc).DeleteMaterial
		},
	}
}
