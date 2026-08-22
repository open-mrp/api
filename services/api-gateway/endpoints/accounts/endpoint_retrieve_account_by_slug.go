package accountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to look up an account by portal slug.
type RetrieveAccountBySlugRequest struct {
	// Portal slug of the account to look up.
	Slug string `path:"slug" validate:"required"`
}

// Returns a minimal public profile for the account that owns the given portal slug.
//
// This endpoint does not require authentication; it is intended for customer portal branding lookups. The logo and favicon are returned as download URLs that stay valid for one hour.
type RetrieveAccountBySlugEndpoint struct{}

func (e *RetrieveAccountBySlugEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAccountBySlugRequest, *apiresource.PublicAccount] {
	return (&apiendpoint.APIEndpoint[*RetrieveAccountBySlugRequest, *apiresource.PublicAccount]{
		Title:             "Retrieve Account by Slug",
		Method:            http.MethodGet,
		Route:             "/v1/settings/branding/{slug}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ObjectType:        constants.ObjectTypePublicAccount,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAccountBySlugRequest) (*apiresource.PublicAccount, *apierror.APIError) {
			return svc.(AccountSvc).GetAccountBySlug
		},
	})
}
