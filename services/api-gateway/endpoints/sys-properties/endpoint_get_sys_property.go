package syspropertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetSysPropertyRequest is the request to retrieve a single system property by ID.
type GetSysPropertyRequest struct {
	// The ID of the system property to retrieve.
	SysPropertyID string `path:"id" validate:"required"`
}

type GetSysPropertyEndpoint struct{}

func (e *GetSysPropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetSysPropertyRequest, *apiresource.SysProperty] {
	return &apiendpoint.APIEndpoint[*GetSysPropertyRequest, *apiresource.SysProperty]{
		Title:             "Get System Property",
		Description:       "Returns a single system property by its ID.",
		Method:            http.MethodGet,
		Route:             "/v1/core/sys-properties/{id}",
		Request:           &GetSysPropertyRequest{},
		Response:          &apiresource.SysProperty{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetSysPropertyRequest) (*apiresource.SysProperty, *apierror.APIError) {
			return svc.(SysPropertySvc).GetSysProperty
		},
	}
}
