package accountgroupep

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

// Request to retrieve an account group.
type RetrieveAccountGroupRequest struct {
	// Account group ID.
	AccountGroupID string `path:"id" validate:"required"`
}

// Returns an account group by ID.
type RetrieveAccountGroupEndpoint struct{}

func (e *RetrieveAccountGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAccountGroupRequest, *apiresource.AccountGroup] {
	return (&apiendpoint.APIEndpoint[*RetrieveAccountGroupRequest, *apiresource.AccountGroup]{
		Title:             "Retrieve Account Group",
		Method:            http.MethodGet,
		Route:             "/v1/sales/account-groups/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCustomerGroups, Action: types.ActionRead}}},
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountGroup,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError) {
			return svc.(AccountGroupSvc).GetAccountGroup
		},
	})
}
