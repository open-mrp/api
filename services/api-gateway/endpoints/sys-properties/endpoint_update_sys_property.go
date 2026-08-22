package syspropertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to update a system property.
type UpdateSysPropertyRequest struct {
	// System property ID.
	SysPropertyID string `path:"id" validate:"required"`
	// The number to move the counter to, so the series carries on from there.
	Value field.Optional[int32] `json:"value,omitzero"`
}

var sampleUpdateSysPropertyValue int32 = 30
var sampleUpdateSysPropertyRequest = &UpdateSysPropertyRequest{
	Value: field.Some(sampleUpdateSysPropertyValue),
}

func (*UpdateSysPropertyRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSysPropertyRequest)
}

// Overrides the value of a system property counter.
//
// Use this to restart or realign a number series, for example to continue the numbering used in a previous system. Records that already carry a number keep it; only the numbers handed out from now on are affected.
type UpdateSysPropertyEndpoint struct{}

func (e *UpdateSysPropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSysPropertyRequest, *apiresource.SysProperty] {
	return (&apiendpoint.APIEndpoint[*UpdateSysPropertyRequest, *apiresource.SysProperty]{
		Title:             "Update System Property",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/settings/properties/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSysProperty,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSystemProperties, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSysPropertyRequest) (*apiresource.SysProperty, *apierror.APIError) {
			return svc.(SysPropertySvc).UpdateSysProperty
		},
	})
}
