package accountgroupep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/validate"
)

// Request to create an account group.
type CreateAccountGroupRequest struct {
	// Display name, unique (case-insensitive) among your account's groups.
	Name string `json:"name" validate:"required,max=255"`
	// How the group is used: `type_group` classifies customers, `pricing_group` scopes pricing. Defaults to `type_group`; fixed once created.
	Type field.Optional[constants.AccountGroupType] `json:"type,omitzero" visibility:"internal"`
	// Whether orders from the group's customers earn commission: `applied` or `exempt`. Defaults to `exempt`.
	CommissionPolicy field.Optional[constants.CommissionPolicy] `json:"commission_policy,omitzero" default:"exempt"`
	// Whether the group's customers are billed for freight: `billed` or `free`. Defaults to `billed`.
	FreightPolicy field.Optional[constants.FreightPolicy] `json:"freight_policy,omitzero" default:"billed"`
	// Values the group's customers inherit when they don't set their own.
	Defaults field.Optional[AccountGroupDefaultsInput] `json:"defaults,omitzero"`
	// IDs of the price lists that apply to every customer in the group, at most 100.
	PriceListIDs []string `json:"price_list_ids,omitzero" validate:"omitempty,max=100"`
	// Free-form description of the group.
	Description field.Optional[string] `json:"description,omitzero"`
}

func init() {
	validate.RegisterWrappedTypes(field.Optional[AccountGroupDefaultsInput]{})
}

// Values an account group's customers inherit when they don't set their own.
type AccountGroupDefaultsInput struct {
	// Business days from an order being issued to it being due to ship, in the `day` unit. Send `null` to use your account default.
	LeadTime field.Clearable[apirequest.QuantityInput] `json:"lead_time,omitzero"`
}

var sampleCreateAccountGroupDescription = "Customers who buy in bulk at wholesale pricing."
var sampleCreateAccountGroupRequest = &CreateAccountGroupRequest{
	Name:             "Wholesale Customers",
	CommissionPolicy: field.Some(constants.CommissionPolicyExempt),
	FreightPolicy:    field.Some(constants.FreightPolicyBilled),
	Description:      field.Some(sampleCreateAccountGroupDescription),
	Defaults:         field.Some(AccountGroupDefaultsInput{LeadTime: field.Set(apirequest.QuantityInput{Value: "5", UnitID: apirequest.DayUnitID})}),
}

func (*CreateAccountGroupRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAccountGroupRequest)
}

// Creates an account group.
type CreateAccountGroupEndpoint struct{}

func (e *CreateAccountGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAccountGroupRequest, *apiresource.AccountGroup] {
	return (&apiendpoint.APIEndpoint[*CreateAccountGroupRequest, *apiresource.AccountGroup]{
		Title:             "Create Account Group",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-groups",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCustomerGroups, Action: types.ActionCreate}}},
		ObjectType:        constants.ObjectTypeAccountGroup,
		IncludeConfig:     accountGroupIncludeConfig(),
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError) {
			return svc.(AccountGroupSvc).CreateAccountGroup
		},
		LocationFunc: func(resp *apiresource.AccountGroup) string {
			return "/v1/sales/account-groups/" + resp.ID
		},
	})
}
