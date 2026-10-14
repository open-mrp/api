package batchreleaseep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve a department's release policy.
type RetrieveReleasePolicyRequest struct {
	// ID of the department.
	DepartmentID string `path:"id" validate:"required"`
}

// Returns the sign-off a department requires before a batch leaves it.
type RetrieveReleasePolicyEndpoint struct{}

func (e *RetrieveReleasePolicyEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveReleasePolicyRequest, *apiresource.ReleasePolicy] {
	return (&apiendpoint.APIEndpoint[*RetrieveReleasePolicyRequest, *apiresource.ReleasePolicy]{
		Title:             "Retrieve Release Policy",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/departments/{id}/release-policy",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeReleasePolicy,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDepartments, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveReleasePolicyRequest) (*apiresource.ReleasePolicy, *apierror.APIError) {
			return svc.(BatchReleaseSvc).GetReleasePolicy
		},
	})
}
