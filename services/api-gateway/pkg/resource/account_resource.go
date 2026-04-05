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

// Account represents a full account with optional branding and portal sub-resources.
type Account struct {
	// The unique identifier for the account.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
	// The display name of the account.
	Name string `json:"name" validate:"required"`
	// The default billing address.
	DefaultBillingAddress *Address `json:"default_billing_address" expandable:"true"`
	// The default shipping address.
	DefaultShippingAddress *Address `json:"default_shipping_address" expandable:"true"`
	// The account branding configuration.
	Branding *AccountBranding `json:"branding" expandable:"true"`
	// The account portal configuration.
	Portal *AccountPortal `json:"portal" expandable:"true"`
	// The timestamp when the account was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the account was last updated.
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

// AccountBranding holds the branding metadata for an account.
type AccountBranding struct {
	// The unique identifier for the branding record.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_branding"`
	// The support email address.
	SupportEmail *string `json:"support_email"`
	// The support phone number.
	PhoneNumber *string `json:"phone_number"`
	// The logo URL (S3 key).
	LogoURL *string `json:"logo_url"`
	// The Facebook handle.
	FacebookHandle *string `json:"facebook_handle"`
	// The Instagram handle.
	InstagramHandle *string `json:"instagram_handle"`
	// The LinkedIn handle.
	LinkedInHandle *string `json:"linkedin_handle"`
	// The Twitter handle.
	TwitterHandle *string `json:"twitter_handle"`
	// The website URL.
	WebsiteURL *string `json:"website_url"`
	// The timestamp when the branding was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the branding was last updated.
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

// AccountPortal holds the portal metadata for an account.
type AccountPortal struct {
	// The unique identifier for the portal record.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_portal"`
	// The portal slug.
	Slug string `json:"slug" validate:"required"`
	// The timestamp when the portal was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the portal was last updated.
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

// PublicAccount is a minimal account representation for unauthenticated slug lookups.
type PublicAccount struct {
	// The unique identifier for the account.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=public_account"`
	// The display name of the account.
	Name string `json:"name" validate:"required"`
	// The portal slug.
	Slug string `json:"slug" validate:"required"`
	// The default billing address.
	DefaultBillingAddress *Address `json:"default_billing_address" expandable:"true"`
	// The support email address.
	SupportEmail *string `json:"support_email"`
	// The logo URL.
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

// AccountLogoURL holds a presigned URL for an account's logo.
type AccountLogoURL struct {
	// The presigned URL for the logo image, or null if no logo exists.
	URL *string `json:"url"`
}

var SampleAccountLogoURL = &AccountLogoURL{}

func (*AccountLogoURL) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountLogoURL)
}

// AccountPhotoUploadResult is the response for a photo upload.
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
