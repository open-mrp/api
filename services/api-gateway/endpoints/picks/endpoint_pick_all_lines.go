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

// Request to mark all lines on a pick as picked.
type PickAllLinesRequest struct {
	// Pick ID.
	PickID string `path:"id" validate:"required"`
}

// Marks all lines on a pick as picked.
//
// Sets each unpacked line's picked quantity to the quantity still outstanding on its sales order line, after accounting for what other pick lines for that order line have already picked, and never lowers a line: calling it again changes nothing, and an over-pick is kept as recorded. Lines that have already been packed, and every line of a finished pick, are unaffected. Use this to fill in a full pick in one call instead of picking each line individually; nothing is shipped until the pick is packed.
type PickAllLinesEndpoint struct{}

func (e *PickAllLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*PickAllLinesRequest, *apiresource.Pick] {
	return (&apiendpoint.APIEndpoint[*PickAllLinesRequest, *apiresource.Pick]{
		Title:             "Pick All Lines",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/picks/{id}/actions/pick",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypePick,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainPicks, Action: types.ActionUpdate},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *PickAllLinesRequest) (*apiresource.Pick, *apierror.APIError) {
			return svc.(PickSvc).PickAllLines
		},
	})
}
