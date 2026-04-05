package partep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeletePartRequest is the request to delete a part.
type DeletePartRequest struct {
	// The item ID of the part to delete.
	ItemID string `path:"id" validate:"required"`
}

type DeletePartEndpoint struct{}

func (e *DeletePartEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeletePartRequest, *apiresource.Part] {
	return &apiendpoint.APIEndpoint[*DeletePartRequest, *apiresource.Part]{
		Title:             "Delete Part",
		Description:       "Soft-deletes a part by setting its deleted_at timestamp.",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/parts/{id}",
		Request:           &DeletePartRequest{},
		Response:          &apiresource.Part{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePartRequest) (*apiresource.Part, *apierror.APIError) {
			return svc.(PartSvc).DeletePart
		},
	}
}
