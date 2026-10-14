package syspropertyep

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

// Request to retrieve a system property by ID.
type RetrieveSysPropertyRequest struct {
	// System property ID.
	SysPropertyID string `path:"id" validate:"required"`
}

// Returns a system property by ID.
type RetrieveSysPropertyEndpoint struct{}

func (e *RetrieveSysPropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveSysPropertyRequest, *apiresource.SysProperty] {
	return (&apiendpoint.APIEndpoint[*RetrieveSysPropertyRequest, *apiresource.SysProperty]{
		Title:             "Retrieve System Property",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/settings/properties/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSysProperty,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainSystemProperties, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveSysPropertyRequest) (*apiresource.SysProperty, *apierror.APIError) {
			return svc.(SysPropertySvc).GetSysProperty
		},
	})
}
