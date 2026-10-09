package accountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to partially update an account.
type UpdateAccountRequest struct {
	// ID of the account to update.
	AccountID string `path:"id" validate:"required"`
	// The account's display name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// The email address customers are directed to for support. Pass null to remove it, as for the other
	// branding fields.
	SupportEmail field.Clearable[string] `json:"support_email,omitzero" validate:"omitempty,custom_email,max=255"`
	// The account's public contact phone number.
	PhoneNumber field.Clearable[string] `json:"phone_number,omitzero" validate:"omitempty,max=255"`
	// URL slug for the account's customer portal.
	//
	// Letters and digits, in runs joined by single hyphens (`acme-inc`); letters are saved lowercase. The slug is unique across all accounts, ignoring case; updating to one that is already taken returns a conflict error. Changing it changes the portal address customers use, so existing portal links stop resolving. An account without a portal gets one at this slug.
	Slug field.Optional[string] `json:"slug,omitzero" validate:"omitempty,min=3,max=255,slug"`
	// The account's public website, as an `http` or `https` URL.
	WebsiteURL field.Clearable[string] `json:"website_url,omitzero" validate:"omitempty,http_url,max=2083"`
	// Facebook handle.
	FacebookHandle field.Clearable[string] `json:"facebook_handle,omitzero" validate:"omitempty,max=255"`
	// Instagram handle.
	InstagramHandle field.Clearable[string] `json:"instagram_handle,omitzero" validate:"omitempty,max=255"`
	// LinkedIn handle.
	LinkedInHandle field.Clearable[string] `json:"linkedin_handle,omitzero" validate:"omitempty,max=255"`
	// Twitter handle.
	TwitterHandle field.Clearable[string] `json:"twitter_handle,omitzero" validate:"omitempty,max=255"`
	// Default billing address for the account's orders. Must be one of the account's own addresses.
	DefaultBillingAddressID field.Optional[string] `json:"default_billing_address_id,omitzero" validate:"omitempty"`
	// Default shipping address for the account's orders. Must be one of the account's own addresses.
	DefaultShippingAddressID field.Optional[string] `json:"default_shipping_address_id,omitzero" validate:"omitempty"`
	// Default billing address saved to the account with the update, in place of `default_billing_address_id`: a new address, or an update to one of the account's own addresses named by its `id`.
	//
	// Saving it needs no permission beyond updating the account.
	DefaultBillingAddress field.Optional[apirequest.InlineAddressInput] `json:"default_billing_address,omitzero"`
	// Default shipping address saved to the account with the update, in place of `default_shipping_address_id`: a new address, or an update to one of the account's own addresses named by its `id`.
	//
	// Saving it needs no permission beyond updating the account. An identical `default_billing_address` and `default_shipping_address` are saved as one address.
	DefaultShippingAddress field.Optional[apirequest.InlineAddressInput] `json:"default_shipping_address,omitzero"`
}

var sampleUpdateAccountRequest = &UpdateAccountRequest{
	Name: field.Some("Acme Inc."),
}

func (*UpdateAccountRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountRequest)
}

// Partially updates an account's name, branding, and portal settings.
//
// Only the fields provided in the request are changed. You can only update the account you are acting in. The logo and favicon are not set here; upload them through their own endpoints.
type UpdateAccountEndpoint struct{}

func (e *UpdateAccountEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountRequest, *apiresource.Account] {
	return (&apiendpoint.APIEndpoint[*UpdateAccountRequest, *apiresource.Account]{
		Title:               "Update Account",
		Method:              http.MethodPatch,
		Route:               "/v1/identity/accounts/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAccount,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountRequest) (*apiresource.Account, *apierror.APIError) {
			return svc.(AccountSvc).UpdateAccount
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccount,
			Fields:     []string{"branding", "portal", "default_billing_address", "default_shipping_address"},
		}),
	})
}
