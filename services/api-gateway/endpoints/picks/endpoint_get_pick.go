package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

type GetPickRequest struct {
	PickID   string   `path:"id" validate:"required"`
	Includes []string `query:"include"`
}

type GetPickEndpoint struct{}

func (e *GetPickEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPickRequest, *apiresource.PickDetail] {
	return &apiendpoint.APIEndpoint[*GetPickRequest, *apiresource.PickDetail]{
		Title:             "Get Pick",
		Description:       "Returns a single pick by its ID.",
		Method:            http.MethodGet,
		Route:             "/v1/operations/picks/{id}",
		Request:           &GetPickRequest{},
		Response:          &apiresource.PickDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetPickRequest) (*apiresource.PickDetail, *apierror.APIError) {
			return svc.(PickSvc).GetPick
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePick,
			Fields:     []string{"lines"},
		}),
	}
}
