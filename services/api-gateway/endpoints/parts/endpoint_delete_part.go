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
	// Part ID.
	ItemID string `path:"id" validate:"required"`
}

type DeletePartEndpoint struct{}

func (e *DeletePartEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeletePartRequest, *apiresource.Part] {
	return &apiendpoint.APIEndpoint[*DeletePartRequest, *apiresource.Part]{
		Title:             "Delete Part",
		Description:       "Deletes a part. Sets the deleted_at timestamp rather than removing the record.",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/catalog/parts/{id}",
		Request:           &DeletePartRequest{},
		Response:          &apiresource.Part{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePartRequest) (*apiresource.Part, *apierror.APIError) {
			return svc.(PartSvc).DeletePart
		},
	}
}
