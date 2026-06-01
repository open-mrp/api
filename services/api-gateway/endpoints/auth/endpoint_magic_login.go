package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to exchange a magic login token for a session.
type MagicLoginRequest struct {
	// Magic login token from the "already registered" email.
	Token string `json:"token" validate:"required" sensitive:"true"` // #nosec G117 - Struct field, not a hardcoded credential
}

var sampleMagicLoginRequest = &MagicLoginRequest{
	Token: apiresource.SampleAccessToken,
}

func (*MagicLoginRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleMagicLoginRequest)
}

// Exchanges a magic login token for a session, setting access and refresh tokens in cookies.
type MagicLoginEndpoint struct{}

func (e *MagicLoginEndpoint) Materialize() *apiendpoint.APIEndpoint[*MagicLoginRequest, *apiresource.User] {
	return (&apiendpoint.APIEndpoint[*MagicLoginRequest, *apiresource.User]{
		Title:             "Magic Login",
		Method:            http.MethodPost,
		Route:             "/v1/auth/actions/magic-login",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ObjectType:        constants.ObjectTypeUser,
		ServiceHandler: func(svc any) func(ctx context.Context, req *MagicLoginRequest) (*apiresource.User, *apierror.APIError) {
			return svc.(AuthSvc).MagicLogin
		},
	})
}
