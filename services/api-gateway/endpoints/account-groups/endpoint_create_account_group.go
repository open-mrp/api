package accountgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to create an account group.
type CreateAccountGroupRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Account group type.
	//
	// Cannot be changed after creation.
	// - `pricing_group`: used for pricing rules, such as a "Preferred" group that receives a special discount.
	// - `type_group`: used to categorize accounts, such as "Consumers" or "Distributors".
	Type constants.AccountGroupType `json:"type" validate:"required"`
	// Commission policy. Defaults to `commission_exempt`.
	//
	// - `commission_exempt`: no commission applies.
	// - `commission_applied`: commission applies; if the account group is within a sales rep's territory, it will be assigned to that rep unless overridden.
	CommissionPolicy field.Optional[constants.CommissionPolicy] `json:"commission_policy,omitzero" default:"commission_exempt"`
	// Freight policy. Defaults to `billed_freight`.
	//
	// - `free_freight`: customers within this group will not have to pay for freight.
	// - `billed_freight`: freight will be applied to any order within this account group, unless overridden elsewhere.
	FreightPolicy field.Optional[constants.FreightPolicy] `json:"freight_policy,omitzero" default:"billed_freight"`
	// Description.
	Description field.Optional[string] `json:"description,omitzero"`
}

var sampleCreateAccountGroupRequest = &CreateAccountGroupRequest{
	Name: "Wholesale Customers",
	Type: constants.AccountGroupTypeTypeGroup,
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
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountGroup,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError) {
			return svc.(AccountGroupSvc).CreateAccountGroup
		},
		LocationFunc: func(resp *apiresource.AccountGroup) string {
			return "/v1/sales/account-groups/" + resp.ID
		},
	})
}
