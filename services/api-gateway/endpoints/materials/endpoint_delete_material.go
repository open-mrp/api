package materialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a material.
type DeleteMaterialRequest struct {
	// Material ID.
	ItemID string `path:"id" validate:"required"`
}

type DeleteMaterialEndpoint struct{}

func (e *DeleteMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteMaterialRequest, *apiresource.Material] {
	return &apiendpoint.APIEndpoint[*DeleteMaterialRequest, *apiresource.Material]{
		Title:             "Delete Material",
		Description:       "Deletes a material by ID.",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/catalog/materials/{id}",
		Request:           &DeleteMaterialRequest{},
		Response:          &apiresource.Material{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteMaterialRequest) (*apiresource.Material, *apierror.APIError) {
			return svc.(MaterialSvc).DeleteMaterial
		},
	}
}
