package syspropertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to update a system property.
type UpdateSysPropertyRequest struct {
	// System property ID.
	SysPropertyID string `path:"id" validate:"required"`
	// Counter value.
	Value *int32 `json:"value,omitempty" nullable:"false"`
}

var sampleUpdateSysPropertyValue int32 = 30
var sampleUpdateSysPropertyRequest = &UpdateSysPropertyRequest{
	Value: &sampleUpdateSysPropertyValue,
}

func (*UpdateSysPropertyRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSysPropertyRequest)
}

type UpdateSysPropertyEndpoint struct{}

func (e *UpdateSysPropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSysPropertyRequest, *apiresource.SysProperty] {
	return &apiendpoint.APIEndpoint[*UpdateSysPropertyRequest, *apiresource.SysProperty]{
		Title:             "Update System Property",
		Description:       "Partially updates the value of a system property.",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/core/sys-properties/{id}",
		Request:           &UpdateSysPropertyRequest{},
		Response:          &apiresource.SysProperty{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSysPropertyRequest) (*apiresource.SysProperty, *apierror.APIError) {
			return svc.(SysPropertySvc).UpdateSysProperty
		},
	}
}
