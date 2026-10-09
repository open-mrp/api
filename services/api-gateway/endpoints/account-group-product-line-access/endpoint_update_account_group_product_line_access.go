package accountgroupproductlineaccessep

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

// Request to update product line access for an account group.
type UpdateAccountGroupProductLineAccessRequest struct {
	// Account group ID.
	AccountGroupID string `path:"account_group_id" validate:"required"`
	// IDs of the product lines the account group should have access to.
	//
	// The provided list replaces the account group's existing set of product lines, and each ID must be a product line your account owns.
	//
	// The list must name at least one product line; an empty list is rejected without changing anything. Use Delete Account Group Product Line Access to revoke the group's access entirely.
	ProductLineIDs field.Optional[[]string] `json:"product_line_ids,omitzero"`
}

var sampleUpdateAccountGroupProductLineAccessRequest = &UpdateAccountGroupProductLineAccessRequest{
	ProductLineIDs: field.Some([]string{apiresource.SampleProductLineID}),
}

func (*UpdateAccountGroupProductLineAccessRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountGroupProductLineAccessRequest)
}

// Replaces the set of product lines accessible to an account group.
//
// This is a full replacement, not a merge: product lines omitted from the request lose access. The account group must already have at least one product line granted, otherwise the request returns a not-found error and the grant has to be made with Create Account Group Product Line Access.
type UpdateAccountGroupProductLineAccessEndpoint struct{}

func (e *UpdateAccountGroupProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountGroupProductLineAccessRequest, *apiresource.AccountGroupProductLineAccess] {
	return (&apiendpoint.APIEndpoint[*UpdateAccountGroupProductLineAccessRequest, *apiresource.AccountGroupProductLineAccess]{
		Title:             "Update Account Group Product Line Access",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/sales/product-line-access/account-groups/{account_group_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountGroupProductLineAccess,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductLineAccess, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountGroupProductLineAccessRequest) (*apiresource.AccountGroupProductLineAccess, *apierror.APIError) {
			return svc.(AccountGroupProductLineAccessSvc).UpdateAccountGroupProductLineAccess
		},
	})
}
