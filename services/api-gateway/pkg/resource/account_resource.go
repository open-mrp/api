package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAccountID = "ac_01gf7a8200eaj8fke1xvw4h50x"
const SampleAccountName = "Acme Inc."
const SampleAccountBrandingID = "abr_01gf7a8200eaj8fke1xvw4h50x"
const SampleAccountPortalID = "apo_01gf7a8200eaj8fke1xvw4h50x"
const SampleAccountPortalSlug = "acme"

// Account with optional branding and portal sub-resources.
type Account struct {
	// Account ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Default billing address.
	DefaultBillingAddress *Address `json:"default_billing_address" expandable:"true"`
	// Default shipping address.
	DefaultShippingAddress *Address `json:"default_shipping_address" expandable:"true"`
	// Branding configuration.
	Branding *AccountBranding `json:"branding" expandable:"true"`
	// Portal configuration.
	Portal *AccountPortal `json:"portal" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAccount = &Account{
	ID:        SampleAccountID,
	Object:    constants.ObjectTypeAccount,
	Name:      SampleAccountName,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Account) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccount)
}

// Branding metadata for an account.
type AccountBranding struct {
	// Branding ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_branding"`
	// Support email address.
	SupportEmail *string `json:"support_email"`
	// Support phone number.
	PhoneNumber *string `json:"phone_number"`
	// Logo URL.
	LogoURL *string `json:"logo_url"`
	// Facebook handle.
	FacebookHandle *string `json:"facebook_handle"`
	// Instagram handle.
	InstagramHandle *string `json:"instagram_handle"`
	// LinkedIn handle.
	LinkedInHandle *string `json:"linkedin_handle"`
	// Twitter handle.
	TwitterHandle *string `json:"twitter_handle"`
	// Website URL.
	WebsiteURL *string `json:"website_url"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAccountBranding = &AccountBranding{
	ID:        SampleAccountBrandingID,
	Object:    constants.ObjectTypeAccountBranding,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AccountBranding) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountBranding)
}

// Portal metadata for an account.
type AccountPortal struct {
	// Portal ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_portal"`
	// Portal slug.
	Slug string `json:"slug" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAccountPortal = &AccountPortal{
	ID:        SampleAccountPortalID,
	Object:    constants.ObjectTypeAccountPortal,
	Slug:      SampleAccountPortalSlug,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AccountPortal) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountPortal)
}

// Minimal account representation for unauthenticated slug lookups.
type PublicAccount struct {
	// Account ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=public_account"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Portal slug.
	Slug string `json:"slug" validate:"required"`
	// Default billing address.
	DefaultBillingAddress *Address `json:"default_billing_address" expandable:"true"`
	// Support email address.
	SupportEmail *string `json:"support_email"`
	// Logo URL.
	LogoURL *string `json:"logo_url"`
}

var SamplePublicAccount = &PublicAccount{
	ID:     SampleAccountID,
	Object: constants.ObjectTypePublicAccount,
	Name:   SampleAccountName,
	Slug:   SampleAccountPortalSlug,
}

func (*PublicAccount) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePublicAccount)
}

// Presigned URL for an account's logo.
type AccountLogoURL struct {
	// Presigned URL. Null if no logo exists.
	URL *string `json:"url"`
}

var SampleAccountLogoURL = &AccountLogoURL{}

func (*AccountLogoURL) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountLogoURL)
}

// Result of an account photo upload.
type AccountPhotoUploadResult struct {
	// Whether the upload was successful.
	Success bool `json:"success"`
}

var SampleAccountPhotoUploadResult = &AccountPhotoUploadResult{
	Success: true,
}

func (*AccountPhotoUploadResult) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountPhotoUploadResult)
}
