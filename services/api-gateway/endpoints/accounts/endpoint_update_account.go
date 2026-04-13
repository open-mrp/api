package accountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to partially update an account.
type UpdateAccountRequest struct {
	// Account ID.
	AccountID string `path:"id" validate:"required"`
	// Display name.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Support email address.
	SupportEmail *string `json:"support_email,omitempty" nullable:"false" validate:"omitnil,custom_email,max=255"`
	// Support phone number.
	PhoneNumber *string `json:"phone_number,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Portal slug.
	Slug *string `json:"slug,omitempty" nullable:"false" validate:"omitempty,min=3,max=255"`
	// Website URL.
	WebsiteURL *string `json:"website_url,omitempty" nullable:"false" validate:"omitempty,url,max=2083"`
	// Facebook handle.
	FacebookHandle *string `json:"facebook_handle,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Instagram handle.
	InstagramHandle *string `json:"instagram_handle,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// LinkedIn handle.
	LinkedInHandle *string `json:"linkedin_handle,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Twitter handle.
	TwitterHandle *string `json:"twitter_handle,omitempty" nullable:"false" validate:"omitempty,max=255"`
}

var sampleUpdateAccountName = "Acme Inc."

var sampleUpdateAccountRequest = &UpdateAccountRequest{
	Name: &sampleUpdateAccountName,
}

func (*UpdateAccountRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountRequest)
}

type UpdateAccountEndpoint struct{}

func (e *UpdateAccountEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountRequest, *apiresource.Account] {
	return &apiendpoint.APIEndpoint[*UpdateAccountRequest, *apiresource.Account]{
		Title:             "Update Account",
		Description:       "Partially updates an account's name, branding, and portal settings.",
		Method:            http.MethodPatch,
		Route:             "/v1/identity/accounts/{id}",
		ContentType:       "application/json",
		Request:           &UpdateAccountRequest{},
		Response:          &apiresource.Account{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountRequest) (*apiresource.Account, *apierror.APIError) {
			return svc.(AccountSvc).UpdateAccount
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccount,
			Fields:     []string{"branding", "portal"},
		}),
	}
}
