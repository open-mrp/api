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

// UpdateAccountRequest is the request to partially update an account.
type UpdateAccountRequest struct {
	// The ID of the account to update.
	AccountID string `path:"id" validate:"required"`
	// The display name of the account.
	Name *string `json:"name,omitempty"`
	// The support email address.
	SupportEmail *string `json:"support_email,omitempty" validate:"omitnil,custom_email" nullable:"true"`
	// The support phone number.
	PhoneNumber *string `json:"phone_number,omitempty" nullable:"true"`
	// The portal slug.
	Slug *string `json:"slug,omitempty" validate:"omitempty,min=3"`
	// The website URL.
	WebsiteURL *string `json:"website_url,omitempty" validate:"omitempty,url" nullable:"true"`
	// The Facebook handle.
	FacebookHandle *string `json:"facebook_handle,omitempty" nullable:"true"`
	// The Instagram handle.
	InstagramHandle *string `json:"instagram_handle,omitempty" nullable:"true"`
	// The LinkedIn handle.
	LinkedInHandle *string `json:"linkedin_handle,omitempty" nullable:"true"`
	// The Twitter handle.
	TwitterHandle *string `json:"twitter_handle,omitempty" nullable:"true"`
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
