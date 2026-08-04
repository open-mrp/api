package accountgroupep

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

// Request to partially update an account group.
type UpdateAccountGroupRequest struct {
	// Account group ID.
	AccountGroupID string `path:"id" validate:"required"`
	// Display name of the account group.
	//
	// Must be unique within your account.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Free-form description of the account group.
	Description field.Clearable[string] `json:"description,omitzero"`
	// How sales commission applies to accounts in this group.
	//
	// - `commission_applied`: sales commission is calculated on orders from accounts in this group.
	// - `commission_exempt`: orders from accounts in this group are exempt from commission.
	CommissionPolicy field.Optional[constants.CommissionPolicy] `json:"commission_policy,omitzero"`
	// How freight charges apply to orders from accounts in this group.
	//
	// - `free_freight`: customers within this group will not have to pay for freight.
	// - `billed_freight`: freight will be applied to any order within this account group, unless overridden elsewhere.
	FreightPolicy field.Optional[constants.FreightPolicy] `json:"freight_policy,omitzero"`
}

var sampleUpdateAccountGroupDescription = "Customers who buy in bulk at wholesale pricing."
var sampleUpdateAccountGroupRequest = &UpdateAccountGroupRequest{
	Name:             field.Some("Updated Wholesale Customers"),
	Description:      field.Set(sampleUpdateAccountGroupDescription),
	CommissionPolicy: field.Some(constants.CommissionPolicyExempt),
	FreightPolicy:    field.Some(constants.FreightPolicyBilled),
}

func (*UpdateAccountGroupRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountGroupRequest)
}

// Partially updates an account group.
//
// Only the provided fields are changed. The account group's `type` cannot be changed after creation, and renaming the group to a name another group in your account already uses returns a conflict error.
//
// A new commission or freight policy takes effect for every account already in the group, not just accounts added afterwards.
type UpdateAccountGroupEndpoint struct{}

func (e *UpdateAccountGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountGroupRequest, *apiresource.AccountGroup] {
	return (&apiendpoint.APIEndpoint[*UpdateAccountGroupRequest, *apiresource.AccountGroup]{
		Title:               "Update Account Group",
		Method:              http.MethodPatch,
		Route:               "/v1/sales/account-groups/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCustomerGroups, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeAccountGroup,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError) {
			return svc.(AccountGroupSvc).UpdateAccountGroup
		},
	})
}
