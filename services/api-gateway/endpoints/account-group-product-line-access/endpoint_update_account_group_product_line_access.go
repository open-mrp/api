package accountgroupproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to update product line access for an account group.
type UpdateAccountGroupProductLineAccessRequest struct {
	// Account group ID.
	AccountGroupID string `path:"account_group_id" validate:"required"`
	// IDs of the product lines the account group should have access to.
	//
	// The provided list replaces the account group's existing set of product lines; every ID must belong to your account.
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
// This is a full replacement, not a merge: product lines omitted from the request lose access.
type UpdateAccountGroupProductLineAccessEndpoint struct{}

func (e *UpdateAccountGroupProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountGroupProductLineAccessRequest, *apiresource.AccountGroupProductLineAccess] {
	return (&apiendpoint.APIEndpoint[*UpdateAccountGroupProductLineAccessRequest, *apiresource.AccountGroupProductLineAccess]{
		Title:             "Update Account Group Product Line Access",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/sales/product-line-access/account-groups/{account_group_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
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
