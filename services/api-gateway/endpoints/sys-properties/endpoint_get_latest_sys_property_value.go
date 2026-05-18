package syspropertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get the latest value for a system property type.
type GetLatestSysPropertyValueRequest struct {
	// System property type code (e.g. transaction_number).
	TypeCode string `path:"type_code" validate:"required"`
}

// Returns the next available counter value for a system property type, initializing it if it does not exist for the account.
type GetLatestSysPropertyValueEndpoint struct{}

func (e *GetLatestSysPropertyValueEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetLatestSysPropertyValueRequest, *apiresource.SysPropertyValue] {
	return (&apiendpoint.APIEndpoint[*GetLatestSysPropertyValueRequest, *apiresource.SysPropertyValue]{
		Title:             "Get Latest System Property Value",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/sys-properties/{type_code}/latest-value",
		Request:           &GetLatestSysPropertyValueRequest{},
		Response:          &apiresource.SysPropertyValue{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetLatestSysPropertyValueRequest) (*apiresource.SysPropertyValue, *apierror.APIError) {
			return svc.(SysPropertySvc).GetLatestSysPropertyValue
		},
	}).WithDocSource(e)
}
