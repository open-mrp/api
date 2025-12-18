package authep

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/validate"
)

// The request to login a user
type LoginRequest struct {
	// The username or email of the user
	Identifier string `json:"identifier" validate:"required"`
	// The password of the user
	Password string `json:"password" validate:"required"`
}

func (lr *LoginRequest) Validate() error {
	v := validate.New()

	validate.ValidateUsernameOrEmail(v, lr.Identifier)
	validate.ValidatePasswordPlaintext(v, lr.Password)

	if !v.Valid() {
		var errorMessages []string
		for field, message := range v.Errors {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", field, message))
		}
		return contracts.NewValidationError(strings.Join(errorMessages, "; "))
	}

	return nil
}

var sampleLoginRequest = &LoginRequest{
	Identifier: apiresource.SampleUserUsername,
	Password:   apiresource.SampleUserPassword,
}

func (*LoginRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleLoginRequest)
}

// The current account in use
type CurrentAccount struct {
	// The ID of the current account
	ID string `json:"id" validate:"required"`
}

const loginEndpointDescription = `This endpoint is utilized to login a user. Once completed, the user object is 
returned. An access and refresh token are set in cookies. Learn more about authentication and authorization in our 
[documentation](https://docs.augno.com/guides/authentication).
`

type LoginEndpoint struct {
	apiendpoint.APIEndpoint[*LoginRequest, *apiresource.User]

	group    *apiendpoint.APIEndpointGroup
	service  AuthCtrl
	platform constants.PlatformMode
	bindOnce sync.Once
	handler  http.HandlerFunc
}

func (e *LoginEndpoint) Materialize() apiendpoint.APIEndpointer {
	e.APIEndpoint = apiendpoint.APIEndpoint[*LoginRequest, *apiresource.User]{
		Title:             "Login User",
		Description:       loginEndpointDescription,
		Method:            http.MethodPost,
		Route:             "/v1/auth/actions/login",
		ContentType:       "application/json",
		Request:           &LoginRequest{},
		Response:          &apiresource.User{},
		SuccessStatusCode: http.StatusOK,
		IsPublic:          true,
		Handler: func(ctrl any) apiendpoint.HandlerFunc[
			*LoginRequest, *apiresource.User,
		] {
			return apiendpoint.HandlerFunc[
				*LoginRequest, *apiresource.User,
			](func(ctx context.Context, req *LoginRequest) (*apiresource.User, *contracts.APIError) {
				return ctrl.(AuthCtrl).Login(ctx, req)
			})
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
	return e
}

func (e *LoginEndpoint) GetHandler() http.HandlerFunc {
	e.bindOnce.Do(func() {
		be := apiendpoint.Bind(e.APIEndpoint, e.service)
		e.handler = httptransport.ConvertToHTTPHandler(be)
	})
	return e.handler
}

func (e *LoginEndpoint) WithGroup(g *apiendpoint.APIEndpointGroup, service AuthCtrl, platform constants.PlatformMode) *LoginEndpoint {
	e.group = g
	e.service = service
	e.platform = platform
	return e
}
