package authep

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	"github.com/augno/api/services/api-gateway/internal/middleware"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/validate"
)

// The request to update a user's password
type UpdatePasswordRequest struct {
	// The old password of the user
	OldPassword string `json:"old_password" validate:"required"`
	// The new password of the user
	NewPassword string `json:"new_password" validate:"required"`
}

func (lr *UpdatePasswordRequest) Validate() error {
	v := validate.New()

	validate.ValidatePasswordPlaintext(v, lr.OldPassword)
	validate.ValidatePasswordPlaintext(v, lr.NewPassword)

	if !v.Valid() {
		var errorMessages []string
		for field, message := range v.Errors {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", field, message))
		}
		return contracts.NewValidationError(strings.Join(errorMessages, "; "))
	}

	return nil
}

var sampleUpdatePasswordRequest = &UpdatePasswordRequest{
	OldPassword: apiresource.SampleUserPassword,
	NewPassword: apiresource.SampleNewUserPassword,
}

func (*UpdatePasswordRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePasswordRequest)
}

const updatePasswordEndpointDescription = `This endpoint is utilized to update a user's password. 
Once completed, the user object is returned. Learn more about authentication and authorization in 
our [documentation](https://docs.augno.com/guides/authentication).
`

type UpdatePasswordEndpoint struct {
	apiendpoint.APIEndpoint[*UpdatePasswordRequest, *apiresource.EmptyResource]

	group          *apiendpoint.APIEndpointGroup
	service        AuthCtrl
	platform       constants.PlatformMode
	authMiddleware func(http.HandlerFunc) http.HandlerFunc
	bindOnce       sync.Once
	handler        http.HandlerFunc
}

func (e *UpdatePasswordEndpoint) Materialize() apiendpoint.APIEndpointer {
	e.APIEndpoint = apiendpoint.APIEndpoint[*UpdatePasswordRequest, *apiresource.EmptyResource]{
		Title:             "Update Password",
		Description:       updatePasswordEndpointDescription,
		Method:            http.MethodPut,
		Route:             "/v1/auth/passwords",
		ContentType:       "application/json",
		Request:           &UpdatePasswordRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		IsPublic:          true,
		Handler: func(ctrl any) apiendpoint.HandlerFunc[
			*UpdatePasswordRequest, *apiresource.EmptyResource,
		] {
			return apiendpoint.HandlerFunc[
				*UpdatePasswordRequest, *apiresource.EmptyResource,
			](func(ctx context.Context, req *UpdatePasswordRequest) (*apiresource.EmptyResource, *contracts.APIError) {
				return ctrl.(AuthCtrl).UpdatePassword(ctx, req)
			})
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
	return e
}

func (e *UpdatePasswordEndpoint) GetHandler() http.HandlerFunc {
	e.bindOnce.Do(func() {
		be := apiendpoint.Bind(e.APIEndpoint, e.service)
		e.handler = httptransport.ConvertToHTTPHandler(be)
		if e.authMiddleware != nil {
			e.handler = e.authMiddleware(e.handler)
		}
	})
	return e.handler
}

func (e *UpdatePasswordEndpoint) WithGroup(g *apiendpoint.APIEndpointGroup, service AuthCtrl, platform constants.PlatformMode, authClient *grpcclient.AuthServiceClient) *UpdatePasswordEndpoint {
	e.group = g
	e.service = service
	e.platform = platform

	// Configure AuthMiddleware for this endpoint
	if authClient != nil {
		authMiddlewareConfig := middleware.AuthMiddlewareConfig{
			AuthClient: authClient,
		}
		e.authMiddleware = middleware.AuthMiddleware(authMiddlewareConfig)
	}

	return e
}
