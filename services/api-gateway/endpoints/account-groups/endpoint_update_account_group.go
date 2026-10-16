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
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to partially update an account group.
type UpdateAccountGroupRequest struct {
	// Account group ID.
	AccountGroupID string `path:"id" validate:"required"`
	// Display name, unique (case-insensitive) among your account's groups.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Free-form description of the group. Send `null` to clear it.
	Description field.Clearable[string] `json:"description,omitzero"`
	// Whether orders from the group's customers earn commission: `applied` or `exempt`.
	CommissionPolicy field.Optional[constants.CommissionPolicy] `json:"commission_policy,omitzero"`
	// Whether the group's customers are billed for freight: `billed` or `free`.
	FreightPolicy field.Optional[constants.FreightPolicy] `json:"freight_policy,omitzero"`
	// Values the group's customers inherit. Only fields you send are changed.
	Defaults field.Optional[AccountGroupDefaultsInput] `json:"defaults,omitzero"`
	// IDs of the price lists that apply to every customer in the group, at most 100. Replaces the current set; send `[]` for none.
	PriceListIDs field.Optional[[]string] `json:"price_list_ids,omitzero" validate:"omitempty,max=100"`
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

// Updates an account group.
//
// Only fields you send are changed. New policies apply to every customer already in the group.
type UpdateAccountGroupEndpoint struct{}

func (e *UpdateAccountGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountGroupRequest, *apiresource.AccountGroup] {
	return (&apiendpoint.APIEndpoint[*UpdateAccountGroupRequest, *apiresource.AccountGroup]{
		Title:             "Update Account Group",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/account-groups/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCustomerGroups, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeAccountGroup,
		IncludeConfig:     accountGroupIncludeConfig(),
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError) {
			return svc.(AccountGroupSvc).UpdateAccountGroup
		},
	})
}
