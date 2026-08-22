package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to mark a pick line as picked.
type PickPickLineRequest struct {
	// Pick ID.
	PickID string `path:"pick_id" validate:"required"`
	// Pick line ID.
	PickLineID string `path:"id" validate:"required"`
}

// Marks a pick line as fully picked.
//
// Sets the line's picked quantity to its sales order line's ordered quantity less everything already picked for that order line, including whatever this line had picked before the call. To record a short pick instead, set the quantity yourself with Update Pick Line. Has no effect on a line that has already been packed.
type PickPickLineEndpoint struct{}

func (e *PickPickLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*PickPickLineRequest, *apiresource.PickLine] {
	return (&apiendpoint.APIEndpoint[*PickPickLineRequest, *apiresource.PickLine]{
		Title:             "Pick Pick Line",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/picks/{pick_id}/lines/{id}/actions/pick",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypePickLine,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainPicks, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *PickPickLineRequest) (*apiresource.PickLine, *apierror.APIError) {
			return svc.(PickSvc).PickPickLine
		},
	})
}
