package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// The seller's identity as presented inside a signed-in customer portal: display name, branding, and letterhead address.
//
// This is the counterpart to the public branding profile used on pre-login pages, and it additionally carries the seller's letterhead address for rendering order documents.
type PortalProfile struct {
	// Account ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=portal_profile"`
	// The seller's display name.
	Name string `json:"name" validate:"required"`
	// The URL slug that identifies the seller's customer portal.
	Slug string `json:"slug" validate:"required"`
	// Download URL for the seller's logo, valid for one hour after the response is generated.
	LogoURL *string `json:"logo_url"`
	// Download URL for the seller's customer-portal favicon, valid for one hour after the response is generated.
	FaviconURL *string `json:"favicon_url"`
	// The email address customers are directed to for support.
	SupportEmail *string `json:"support_email"`
	// The seller's letterhead address, shown on customer-facing documents.
	//
	// This is the seller account's own default billing address.
	Address *Address `json:"address"`
}

var samplePortalProfile = &PortalProfile{
	ID:           SampleAccountID,
	Object:       constants.ObjectTypePortalProfile,
	Name:         "Acme Inc",
	Slug:         SampleAccountPortalSlug,
	LogoURL:      nil,
	SupportEmail: nil,
	Address:      SampleAddress,
}

func (*PortalProfile) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(samplePortalProfile)
}
