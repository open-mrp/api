package workcenterep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to delete a work center.
type DeleteWorkCenterRequest struct {
	// ID of the work center to delete.
	WorkCenterID string `path:"id" validate:"required"`
}

// Deletes a work center.
//
// A work center that machines, scanning stations, production steps or account users still belong to can't be deleted (409 `resource_in_use`): move them to another work center first.
type DeleteWorkCenterEndpoint struct{}

func (e *DeleteWorkCenterEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteWorkCenterRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteWorkCenterRequest, *object.Deleted]{
		Title:             "Delete Work Center",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/work-centers/{id}",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeWorkCenter,
		Public:            true,
		AgentTool:         true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteWorkCenterRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(WorkCenterSvc).DeleteWorkCenter, constants.ObjectTypeWorkCenter, "id")
		},
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainDepartments, Action: types.ActionDelete},
		}},
	})
}
