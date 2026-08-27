package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleAccountID = "ac_ykxoradjoeb3"
const SampleAccountName = "Acme Inc."
const SampleAccountBrandingID = "abr_2rygb4fof28b"
const SampleAccountPortalID = "apo_u2esi5el78uv"
const SampleAccountPortalSlug = "acme"

// An organization on OpenMRP, including its branding and customer portal sub-resources.
//
// Your own account and any customer or supplier account you trade with are both represented by this object.
type Account struct {
	// Account ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
	// The account's display name.
	Name string `json:"name" validate:"required"`
	// The address billed by default on orders for this account.
	DefaultBillingAddress *Address `json:"default_billing_address" expandable:"true"`
	// The address shipped to by default on orders for this account.
	DefaultShippingAddress *Address `json:"default_shipping_address" expandable:"true"`
	// Customer-facing branding for the account, such as the logo, support contacts, and social links.
	Branding *AccountBranding `json:"branding" expandable:"true"`
	// The account's customer portal settings, including the portal URL slug.
	Portal *AccountPortal `json:"portal" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAccount = &Account{
	ID:                     SampleAccountID,
	Object:                 constants.ObjectTypeAccount,
	Name:                   SampleAccountName,
	DefaultBillingAddress:  SampleAddress,
	DefaultShippingAddress: SampleAddress,
	Branding:               SampleAccountBranding,
	Portal:                 SampleAccountPortal,
	CreatedAt:              timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:              timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Account) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccount)
}

// The customer-facing branding an account presents on its portal, emails, and documents.
type AccountBranding struct {
	// Branding ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_branding"`
	// The email address customers are directed to for support.
	SupportEmail *string `json:"support_email"`
	// The account's public contact phone number.
	PhoneNumber *string `json:"phone_number"`
	// Stored location of the account's logo image.
	//
	// Logos uploaded through the API are stored as an object key rather than a fetchable link, so use the Get Account Logo URL endpoint to obtain a download URL.
	LogoURL *string `json:"logo_url"`
	// Stored location of the account's customer-portal favicon.
	//
	// Favicons uploaded through the API are stored as an object key rather than a fetchable link, so use the Get Account Favicon URL endpoint to obtain a download URL.
	FaviconURL *string `json:"favicon_url"`
	// Facebook handle.
	FacebookHandle *string `json:"facebook_handle"`
	// Instagram handle.
	InstagramHandle *string `json:"instagram_handle"`
	// LinkedIn handle.
	LinkedInHandle *string `json:"linkedin_handle"`
	// Twitter handle.
	TwitterHandle *string `json:"twitter_handle"`
	// The account's public website.
	WebsiteURL *string `json:"website_url"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAccountBranding = &AccountBranding{
	ID:              SampleAccountBrandingID,
	Object:          constants.ObjectTypeAccountBranding,
	SupportEmail:    new("support@acme.example.com"),
	PhoneNumber:     new("+1-614-555-0100"),
	LogoURL:         new("https://cdn.openmrp.ai/branding/abr_2rygb4fof28b/logo.png"),
	FaviconURL:      new("https://cdn.openmrp.ai/branding/abr_2rygb4fof28b/favicon.png"),
	FacebookHandle:  new("acmeinc"),
	InstagramHandle: new("acmeinc"),
	LinkedInHandle:  new("acme-inc"),
	TwitterHandle:   new("acmeinc"),
	WebsiteURL:      new("https://www.acme.example.com"),
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AccountBranding) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountBranding)
}

// The customer portal an account publishes for its customers to sign in to.
type AccountPortal struct {
	// Portal ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_portal"`
	// URL slug that identifies the account's customer portal.
	//
	// Unique across all accounts.
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

// The publicly readable branding profile of an account, used to render customer portal pages before anyone signs in.
type PublicAccount struct {
	// Account ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=public_account"`
	// The account's display name.
	Name string `json:"name" validate:"required"`
	// The URL slug that identifies the account's customer portal.
	Slug string `json:"slug" validate:"required"`
	// The address billed by default on orders for this account.
	DefaultBillingAddress *Address `json:"default_billing_address" expandable:"true"`
	// The email address customers are directed to for support.
	SupportEmail *string `json:"support_email"`
	// Stable public CDN URL for the account's logo, safe to cache and embed.
	LogoURL *string `json:"logo_url"`
	// The account's custom portal domain (e.g. shop.acme.com).
	//
	// A custom domain only appears here once it has passed verification; until then the portal is served from its slug URL.
	PortalDomain *string `json:"portal_domain"`
	// Stable public CDN URL for the account's customer-portal favicon, safe to cache and embed.
	FaviconURL *string `json:"favicon_url"`
}

var SamplePublicAccount = &PublicAccount{
	ID:                    SampleAccountID,
	Object:                constants.ObjectTypePublicAccount,
	Name:                  SampleAccountName,
	Slug:                  SampleAccountPortalSlug,
	DefaultBillingAddress: SampleAddress,
	SupportEmail:          new("support@acme.example.com"),
	LogoURL:               new("https://cdn.openmrp.ai/branding/abr_2rygb4fof28b/logo.png"),
	PortalDomain:          new("shop.acme.com"),
	FaviconURL:            new("https://cdn.openmrp.ai/branding/abr_2rygb4fof28b/favicon.png"),
}

func (*PublicAccount) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePublicAccount)
}

// Download URL for an account's logo.
type AccountLogoURL struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_logo_url"`
	// Stable public CDN URL for downloading the account's logo.
	//
	// Safe to cache and embed. No URL is returned when the account has never uploaded a logo or the stored image is no longer available.
	URL *string `json:"url"`
}

var SampleAccountLogoURL = &AccountLogoURL{
	Object: constants.ObjectTypeAccountLogoURL,
	URL:    new("https://cdn.openmrp.ai/branding/abr_2rygb4fof28b/logo.png"),
}

func (*AccountLogoURL) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountLogoURL)
}

// Result of an account logo upload.
type AccountPhotoUploadResult struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_photo_upload_result"`
	// Whether the upload was successful.
	Success bool `json:"success"`
}

var SampleAccountPhotoUploadResult = &AccountPhotoUploadResult{
	Object:  constants.ObjectTypeAccountPhotoUploadResult,
	Success: true,
}

func (*AccountPhotoUploadResult) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountPhotoUploadResult)
}

// Download URL for an account's customer-portal favicon.
type AccountFaviconURL struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_favicon_url"`
	// Stable public CDN URL for downloading the account's favicon.
	//
	// Safe to cache and embed. No URL is returned when the account has never uploaded a favicon or the stored image is no longer available.
	URL *string `json:"url"`
}

var SampleAccountFaviconURL = &AccountFaviconURL{
	Object: constants.ObjectTypeAccountFaviconURL,
	URL:    new("https://cdn.openmrp.ai/branding/abr_2rygb4fof28b/favicon.png"),
}

func (*AccountFaviconURL) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountFaviconURL)
}
