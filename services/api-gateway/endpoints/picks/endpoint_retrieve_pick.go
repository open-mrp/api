package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a pick.
type RetrievePickRequest struct {
	// Pick ID.
	PickID string `path:"id" validate:"required"`
}

// Returns a pick by ID.
type RetrievePickEndpoint struct{}

func (e *RetrievePickEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrievePickRequest, *apiresource.Pick] {
	return (&apiendpoint.APIEndpoint[*RetrievePickRequest, *apiresource.Pick]{
		Title:             "Retrieve Pick",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/picks/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypePick,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainPicks, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrievePickRequest) (*apiresource.Pick, *apierror.APIError) {
			return svc.(PickSvc).GetPick
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePick,
			Fields: []string{
				"sales_order",
				"customer",
				"departments",
				"lines",
				"lines.sales_order_line",
			},
		}),
	})
}
