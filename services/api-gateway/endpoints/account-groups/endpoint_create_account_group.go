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

// Request to create an account group.
type CreateAccountGroupRequest struct {
	// Display name of the account group.
	//
	// Must be unique within your account.
	Name string `json:"name" validate:"required,max=255"`
	// How this account group will be used.
	//
	// - `pricing_group`: used for pricing rules, such as a "Preferred" group that receives a special discount.
	// - `type_group`: used to categorize accounts, such as "Consumers" or "Distributors".
	//
	// The type cannot be changed after creation.
	Type constants.AccountGroupType `json:"type" validate:"required"`
	// How sales commission applies to accounts in this group.
	//
	// - `commission_applied`: sales commission is calculated on orders from accounts in this group.
	// - `commission_exempt`: orders from accounts in this group are exempt from commission.
	//
	// Leave this out and the group is created commission-exempt, so orders from its accounts earn no sales commission until you change it.
	CommissionPolicy field.Optional[constants.CommissionPolicy] `json:"commission_policy,omitzero" default:"commission_exempt"`
	// How freight charges apply to orders from accounts in this group.
	//
	// - `free_freight`: customers within this group will not have to pay for freight.
	// - `billed_freight`: freight will be applied to any order within this account group, unless overridden elsewhere.
	FreightPolicy field.Optional[constants.FreightPolicy] `json:"freight_policy,omitzero" default:"billed_freight"`
	// Free-form description of the account group.
	Description field.Optional[string] `json:"description,omitzero"`
}

var sampleCreateAccountGroupDescription = "Customers who buy in bulk at wholesale pricing."
var sampleCreateAccountGroupRequest = &CreateAccountGroupRequest{
	Name:             "Wholesale Customers",
	Type:             constants.AccountGroupTypeTypeGroup,
	CommissionPolicy: field.Some(constants.CommissionPolicyExempt),
	FreightPolicy:    field.Some(constants.FreightPolicyBilled),
	Description:      field.Some(sampleCreateAccountGroupDescription),
}

func (*CreateAccountGroupRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAccountGroupRequest)
}

// Creates an account group.
//
// Returns a conflict error if an account group with the same name already exists.
type CreateAccountGroupEndpoint struct{}

func (e *CreateAccountGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAccountGroupRequest, *apiresource.AccountGroup] {
	return (&apiendpoint.APIEndpoint[*CreateAccountGroupRequest, *apiresource.AccountGroup]{
		Title:               "Create Account Group",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/sales/account-groups",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCustomerGroups, Action: types.ActionCreate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeAccountGroup,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError) {
			return svc.(AccountGroupSvc).CreateAccountGroup
		},
		LocationFunc: func(resp *apiresource.AccountGroup) string {
			return "/v1/sales/account-groups/" + resp.ID
		},
	})
}
