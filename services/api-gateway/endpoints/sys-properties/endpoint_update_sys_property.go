package syspropertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to update a system property.
type UpdateSysPropertyRequest struct {
	// System property ID.
	SysPropertyID string `path:"id" validate:"required"`
	// The new counter value, such as the next transaction or document number to assign.
	Value field.Optional[int32] `json:"value,omitzero"`
}

var sampleUpdateSysPropertyValue int32 = 30
var sampleUpdateSysPropertyRequest = &UpdateSysPropertyRequest{
	Value: field.Some(sampleUpdateSysPropertyValue),
}

func (*UpdateSysPropertyRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSysPropertyRequest)
}

// Partially updates the value of a system property.
type UpdateSysPropertyEndpoint struct{}

func (e *UpdateSysPropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSysPropertyRequest, *apiresource.SysProperty] {
	return (&apiendpoint.APIEndpoint[*UpdateSysPropertyRequest, *apiresource.SysProperty]{
		Title:             "Update System Property",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/core/sys-properties/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSysProperty,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSysPropertyRequest) (*apiresource.SysProperty, *apierror.APIError) {
			return svc.(SysPropertySvc).UpdateSysProperty
		},
	})
}
