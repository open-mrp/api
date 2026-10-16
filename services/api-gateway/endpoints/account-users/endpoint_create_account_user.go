package accountuserep

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

// Request to create an account user.
type CreateAccountUserRequest struct {
	// The person's name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// The person's email address.
	//
	// When someone with this email already has an OpenMRP login, they're added to the account under it. 409 if they're already a member.
	Email field.Optional[string] `json:"email,omitzero" validate:"omitempty,custom_email,max=255"`
	// A scanning-station login name, for a station without an email. Stations are moving to devices.
	Username field.Optional[string] `json:"username,omitzero" validate:"omitempty,username" visibility:"internal"`
	// The scanning-station login's password, required with `username` and refused otherwise.
	Password field.Optional[string] `json:"password,omitzero" validate:"omitempty,password" sensitive:"true" visibility:"internal"` // #nosec G117 -- API request field for user password input
	// ID of the role that sets what the person can do.
	//
	// A customer's contacts always get the customer portal role. A role of type `sales_rep` becomes the account's sales-rep role.
	RoleID field.Optional[string] `json:"role_id,omitzero"`
	// ID of the work center the person works in.
	WorkCenterID field.Optional[string] `json:"work_center_id,omitzero"`
	// Whether the person can be named as a sales rep: `eligible` or `ineligible` (the default).
	//
	// Members with a sales-rep role are always eligible; scanner and agent roles can't be.
	CommissionEligibility field.Optional[constants.CommissionEligibility] `json:"commission_eligibility,omitzero"`
	// Notification toggles for a customer's or supplier's contact.
	Preferences []NotificationPreferenceItem `json:"preferences,omitzero" visibility:"internal"`
}

var sampleCreateAccountUserRequest = &CreateAccountUserRequest{
	Name:                  field.Some(apiresource.SampleUserName),
	Email:                 field.Some(apiresource.SampleUserEmail),
	RoleID:                field.Some(apiresource.SampleRoleID),
	WorkCenterID:          field.Some(apiresource.SampleWorkCenterID),
	CommissionEligibility: field.Some(constants.CommissionEligibilityIneligible),
}

func (*CreateAccountUserRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAccountUserRequest)
}

// Adds a person to the account you are acting in.
//
// A member of your own team is emailed a sign-in link and takes a seat; the request is refused at your plan's seat limit. A customer's or supplier's contact gets no access and no email until you invite them. A person you removed before is restored rather than added twice. You can add contacts to a customer or supplier only while you manage its users: one with its own OpenMRP subscription, or one another account also deals with, manages its own.
type CreateAccountUserEndpoint struct{}

func (e *CreateAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAccountUserRequest, *apiresource.AccountUser] {
	return (&apiendpoint.APIEndpoint[*CreateAccountUserRequest, *apiresource.AccountUser]{
		Title:             "Create Account User",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/identity/account-users",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainTeamUsers, Action: types.ActionCreate}}, Counterparty: authz.Counterparties(types.ActionCreate)},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError) {
			return svc.(AccountUserSvc).CreateAccountUser
		},
		LocationFunc: func(resp *apiresource.AccountUser) string {
			return "/v1/identity/account-users/" + resp.ID
		},
		ObjectType:    constants.ObjectTypeAccountUser,
		IncludeConfig: accountUserIncludes(),
	})
}
