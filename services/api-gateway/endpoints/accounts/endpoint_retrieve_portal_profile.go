package accountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve a seller's portal profile by slug.
type RetrievePortalProfileRequest struct {
	// Portal slug of the seller whose profile is being retrieved.
	Slug string `path:"slug" validate:"required"`
}

// Returns the seller portal profile for the given slug: the seller's identity and its public letterhead address.
//
// Unlike the public branding lookup, this endpoint requires an authenticated caller. It is used by the logged-in customer portal (e.g. order documents) where the seller's letterhead address is shown.
type RetrievePortalProfileEndpoint struct{}

func (e *RetrievePortalProfileEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrievePortalProfileRequest, *apiresource.PortalProfile] {
	return (&apiendpoint.APIEndpoint[*RetrievePortalProfileRequest, *apiresource.PortalProfile]{
		Title:             "Retrieve Portal Profile",
		Method:            http.MethodGet,
		Route:             "/v1/settings/portal-profiles/{slug}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ObjectType:        constants.ObjectTypePortalProfile,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrievePortalProfileRequest) (*apiresource.PortalProfile, *apierror.APIError) {
			return svc.(AccountSvc).GetPortalProfileBySlug
		},
	})
}
