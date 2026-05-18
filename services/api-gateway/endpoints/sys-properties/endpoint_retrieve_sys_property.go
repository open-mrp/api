package syspropertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
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
		Route:             "/v1/core/sys-properties/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveSysPropertyRequest) (*apiresource.SysProperty, *apierror.APIError) {
			return svc.(SysPropertySvc).GetSysProperty
		},
	})
}
