package accountep

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

// Request to partially update an account.
type UpdateAccountRequest struct {
	// Account ID.
	AccountID string `path:"id" validate:"required"`
	// Display name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Support email address.
	SupportEmail field.Optional[string] `json:"support_email,omitzero" validate:"omitempty,custom_email,max=255"`
	// Support phone number.
	PhoneNumber field.Optional[string] `json:"phone_number,omitzero" validate:"omitempty,max=255"`
	// Portal slug.
	Slug field.Optional[string] `json:"slug,omitzero" validate:"omitempty,min=3,max=255"`
	// Website URL.
	WebsiteURL field.Optional[string] `json:"website_url,omitzero" validate:"omitempty,url,max=2083"`
	// Facebook handle.
	FacebookHandle field.Optional[string] `json:"facebook_handle,omitzero" validate:"omitempty,max=255"`
	// Instagram handle.
	InstagramHandle field.Optional[string] `json:"instagram_handle,omitzero" validate:"omitempty,max=255"`
	// LinkedIn handle.
	LinkedInHandle field.Optional[string] `json:"linkedin_handle,omitzero" validate:"omitempty,max=255"`
	// Twitter handle.
	TwitterHandle field.Optional[string] `json:"twitter_handle,omitzero" validate:"omitempty,max=255"`
}

var sampleUpdateAccountRequest = &UpdateAccountRequest{
	Name: field.Some("Acme Inc."),
}

func (*UpdateAccountRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountRequest)
}

// Partially updates an account's name, branding, and portal settings.
type UpdateAccountEndpoint struct{}

func (e *UpdateAccountEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountRequest, *apiresource.Account] {
	return (&apiendpoint.APIEndpoint[*UpdateAccountRequest, *apiresource.Account]{
		Title:             "Update Account",
		Method:            http.MethodPatch,
		Route:             "/v1/identity/accounts/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccount,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountRequest) (*apiresource.Account, *apierror.APIError) {
			return svc.(AccountSvc).UpdateAccount
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccount,
			Fields:     []string{"branding", "portal"},
		}),
	})
}
