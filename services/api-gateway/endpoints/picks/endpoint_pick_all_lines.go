package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// PickAllLinesRequest is the request to mark all lines on a pick as picked.
type PickAllLinesRequest struct {
	// Pick ID.
	PickID string `path:"id" validate:"required"`
}

type PickAllLinesEndpoint struct{}

func (e *PickAllLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*PickAllLinesRequest, *apiresource.PickDetail] {
	return &apiendpoint.APIEndpoint[*PickAllLinesRequest, *apiresource.PickDetail]{
		Title:             "Pick All Lines",
		Description:       "Marks all lines on a pick as picked.",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/picks/{id}/actions/pick",
		Request:           &PickAllLinesRequest{},
		Response:          &apiresource.PickDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *PickAllLinesRequest) (*apiresource.PickDetail, *apierror.APIError) {
			return svc.(PickSvc).PickAllLines
		},
	}
}
