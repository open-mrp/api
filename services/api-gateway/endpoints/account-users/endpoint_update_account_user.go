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

// Request to update an account user.
type UpdateAccountUserRequest struct {
	// ID of the account user to update.
	AccountUserID string `path:"id" validate:"required"`
	// The person's name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// The person's email address. 409 if another person already uses it.
	Email field.Optional[string] `json:"email,omitzero" validate:"omitempty,custom_email,max=255"`
	// A scanning-station login name. 409 if another login already uses it.
	Username field.Optional[string] `json:"username,omitzero" validate:"omitempty,username" visibility:"internal"`
	// ID of the role that sets what the person can do. Send `null` to clear it.
	RoleID field.Clearable[string] `json:"role_id,omitzero" validate:"omitempty"`
	// ID of the work center the person works in. Send `null` to clear it.
	WorkCenterID field.Clearable[string] `json:"work_center_id,omitzero" validate:"omitempty"`
	// Whether the person can be named as a sales rep: `eligible` or `ineligible`.
	//
	// A member with a sales-rep role stays eligible; scanner and agent roles can't be.
	CommissionEligibility field.Optional[constants.CommissionEligibility] `json:"commission_eligibility,omitzero"`
	// Notification toggles for a customer's or supplier's contact; types left out are unchanged.
	Preferences []NotificationPreferenceItem `json:"preferences,omitzero" visibility:"internal"`
}

var sampleUpdateAccountUserRequest = &UpdateAccountUserRequest{
	Name:                  field.Some(apiresource.SampleUserName),
	RoleID:                field.Set(apiresource.SampleRoleID),
	WorkCenterID:          field.Set(apiresource.SampleWorkCenterID),
	CommissionEligibility: field.Some(constants.CommissionEligibilityEligible),
}

func (*UpdateAccountUserRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountUserRequest)
}

// Updates an account user.
//
// Only fields you send are changed. `name` and `email` belong to the person's profile, which is shared across every account they belong to.
type UpdateAccountUserEndpoint struct{}

func (e *UpdateAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountUserRequest, *apiresource.AccountUser] {
	return (&apiendpoint.APIEndpoint[*UpdateAccountUserRequest, *apiresource.AccountUser]{
		Title:             "Update Account User",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/identity/account-users/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainTeamUsers, Action: types.ActionUpdate}}, Counterparty: authz.Counterparties(types.ActionUpdate)},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError) {
			return svc.(AccountUserSvc).UpdateAccountUser
		},
		ObjectType:    constants.ObjectTypeAccountUser,
		IncludeConfig: accountUserIncludes(),
	})
}
