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

// The request to register a new user
type RegisterRequest struct {
	// The email address for the new user
	Email string `json:"email" validate:"required" example:"jdoe@augno.com"`
	// The password for the new user
	Password string `json:"password" validate:"required" example:"super-secret-password"`
	// The full name of the new user
	Name string `json:"name" validate:"required" example:"John Doe"`
}

func (rr *RegisterRequest) Validate() error {
	v := validate.New()

	validate.ValidateEmail(v, rr.Email)
	validate.ValidatePasswordPlaintext(v, rr.Password)

	if !v.Valid() {
		var errorMessages []string
		for field, message := range v.Errors {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", field, message))
		}
		return contracts.NewValidationError(strings.Join(errorMessages, "; "))
	}

	return nil
}

var sampleRegisterRequest = &RegisterRequest{
	Email:    apiresource.SampleUserEmail,
	Password: apiresource.SampleUserPassword,
	Name:     apiresource.SampleUserName,
}

func (*RegisterRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRegisterRequest)
}

const registerEndpointDescription = `This endpoint is utilized to register a new user. Once completed, the user object is 
returned. An access and refresh token are set in cookies. Learn more about authentication and authorization in our 
[documentation](https://docs.augno.com/guides/authentication).
`

type RegisterEndpoint struct {
	apiendpoint.APIEndpoint[*RegisterRequest, *apiresource.User]

	group    *apiendpoint.APIEndpointGroup
	service  AuthCtrl
	platform constants.PlatformMode
	bindOnce sync.Once
	handler  http.HandlerFunc
}

func (e *RegisterEndpoint) Materialize() apiendpoint.APIEndpointer {
	e.APIEndpoint = apiendpoint.APIEndpoint[*RegisterRequest, *apiresource.User]{
		Title:             "Register User",
		Description:       registerEndpointDescription,
		Method:            http.MethodPost,
		Route:             "/v1/auth/users",
		ContentType:       "application/json",
		Request:           &RegisterRequest{},
		Response:          &apiresource.User{},
		SuccessStatusCode: http.StatusOK,
		IsPublic:          true,
		Handler: func(ctrl any) apiendpoint.HandlerFunc[
			*RegisterRequest, *apiresource.User,
		] {
			return apiendpoint.HandlerFunc[
				*RegisterRequest, *apiresource.User,
			](func(ctx context.Context, req *RegisterRequest) (*apiresource.User, *contracts.APIError) {
				return ctrl.(AuthCtrl).Register(ctx, req)
			})
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
	return e
}

func (e *RegisterEndpoint) GetHandler() http.HandlerFunc {
	e.bindOnce.Do(func() {
		be := apiendpoint.Bind(e.APIEndpoint, e.service)
		e.handler = httptransport.ConvertToHTTPHandler(be)
	})
	return e.handler
}

func (e *RegisterEndpoint) WithGroup(g *apiendpoint.APIEndpointGroup, service AuthCtrl, platform constants.PlatformMode) *RegisterEndpoint {
	e.group = g
	e.service = service
	e.platform = platform
	return e
}
