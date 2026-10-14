package pickep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to void a pick line.
type VoidPickLineRequest struct {
	// Pick ID.
	PickID string `path:"pick_id" validate:"required"`
	// Pick line ID.
	PickLineID string `path:"id" validate:"required"`
}

// Voids a pick line, undoing the picking work recorded on it.
//
// Resets the line's picked quantity to zero without deleting the line, so the quantity can be picked again. Returns a validation error if the line has already been packed.
type VoidPickLineEndpoint struct{}

func (e *VoidPickLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*VoidPickLineRequest, *apiresource.PickLine] {
	return (&apiendpoint.APIEndpoint[*VoidPickLineRequest, *apiresource.PickLine]{
		Title:             "Void Pick Line",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/picks/{pick_id}/lines/{id}/actions/void",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypePickLine,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainPicks, Action: types.ActionUpdate},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *VoidPickLineRequest) (*apiresource.PickLine, *apierror.APIError) {
			return svc.(PickSvc).VoidPickLine
		},
	})
}
