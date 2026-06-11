package partep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a part.
type DeletePartRequest struct {
	// ID of the part to delete.
	ItemID string `path:"id" validate:"required"`
}

// Deletes a part.
//
// This is a soft delete: the part is marked deleted and no longer returned by other endpoints, but the record is retained. Deleting an already-deleted part returns an error.
type DeletePartEndpoint struct{}

func (e *DeletePartEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeletePartRequest, *apiresource.Part] {
	return (&apiendpoint.APIEndpoint[*DeletePartRequest, *apiresource.Part]{
		Title:             "Delete Part",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/catalog/parts/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePartRequest) (*apiresource.Part, *apierror.APIError) {
			return svc.(PartSvc).DeletePart
		},
	})
}
