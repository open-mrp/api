package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// CreateAccountUserRequest is the request to create a new account user.
type CreateAccountUserRequest struct {
	// The user's display name.
	Name *string `json:"name" validate:"omitempty,max=255"`
	// The user's email address.
	Email *string `json:"email" validate:"omitnil,custom_email,max=255"`
	// The user's username.
	Username *string `json:"username" validate:"omitempty,max=255"`
	// The user's password.
	Password *string `json:"password"` // #nosec G117 -- API request field for user password input
	// The ID of the role to assign. Expandable.
	RoleID *string `json:"role_id,omitempty" validate:"omitempty,max=191"`
	// The ID of the department to assign. Expandable.
	DepartmentID *string `json:"department_id,omitempty" validate:"omitempty,max=191"`
	// Whether the user is a sales representative.
	IsSalesRep *bool `json:"is_sales_rep,omitempty"`
	// Whether the user receives order acknowledgement notifications.
	ReceivesOrderAcknowledgements bool `json:"receives_order_acknowledgements"`
	// Whether the user receives invoice notifications.
	ReceivesInvoiceNotifications bool `json:"receives_invoice_notifications"`
	// Whether the user receives purchase order submission notifications.
	ReceivesPurchaseOrderSubmissionNotifications bool `json:"receives_purchase_order_submission_notifications"`
}

var sampleCreateAccountUserName = apiresource.SampleUserName
var sampleCreateAccountUserEmail = apiresource.SampleUserEmail
var sampleCreateAccountUserUsername = apiresource.SampleUserUsername
var sampleCreateAccountUserPassword = apiresource.SampleUserPassword
var sampleCreateAccountUserRoleID = apiresource.SampleRoleID
var sampleCreateAccountUserIsSalesRep = false
var sampleCreateAccountUserRequest = &CreateAccountUserRequest{
	Name:                          &sampleCreateAccountUserName,
	Email:                         &sampleCreateAccountUserEmail,
	Username:                      &sampleCreateAccountUserUsername,
	Password:                      &sampleCreateAccountUserPassword,
	RoleID:                        &sampleCreateAccountUserRoleID,
	IsSalesRep:                    &sampleCreateAccountUserIsSalesRep,
	ReceivesOrderAcknowledgements: true,
}

func (*CreateAccountUserRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAccountUserRequest)
}

type CreateAccountUserEndpoint struct{}

func (e *CreateAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAccountUserRequest, *apiresource.AccountUser] {
	return &apiendpoint.APIEndpoint[*CreateAccountUserRequest, *apiresource.AccountUser]{
		Title:             "Create Account User",
		Description:       "Creates a new account user and invites them to the account.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/identity/account-users",
		Request:           &CreateAccountUserRequest{},
		Response:          &apiresource.AccountUser{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		Extras: apiendpoint.APIEndpointExtras{
			ShieldRequestBody: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError) {
			return svc.(AccountUserSvc).CreateAccountUser
		},
		LocationFunc: func(resp *apiresource.AccountUser) string {
			return "/v1/identity/account-users/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountUser,
			Fields:     []string{"role", "department"},
		}),
	}
}
