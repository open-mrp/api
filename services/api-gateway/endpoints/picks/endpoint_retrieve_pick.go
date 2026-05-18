package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

type RetrievePickRequest struct {
	PickID   string   `path:"id" validate:"required"`
	Includes []string `query:"include"`
}

// Returns a pick by ID.
type RetrievePickEndpoint struct{}

func (e *RetrievePickEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrievePickRequest, *apiresource.PickDetail] {
	return (&apiendpoint.APIEndpoint[*RetrievePickRequest, *apiresource.PickDetail]{
		Title:             "Retrieve Pick",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/picks/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrievePickRequest) (*apiresource.PickDetail, *apierror.APIError) {
			return svc.(PickSvc).GetPick
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePick,
			Fields: []string{
				"sales_order",
				"departments",
				"lines",
				"lines.sales_order_line",
			},
		}),
	})
}
