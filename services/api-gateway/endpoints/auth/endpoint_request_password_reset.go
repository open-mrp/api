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

// The request to request a password reset
type RequestPasswordResetRequest struct {
	// The username or email of the account to reset
	Identifier string `json:"identifier" validate:"required"`
	// The account slug (optional)
	AccountSlug *string `json:"account_slug,omitempty"`
}

func (rr *RequestPasswordResetRequest) Validate() error {
	v := validate.New()

	validate.ValidateUsernameOrEmail(v, rr.Identifier)

	if !v.Valid() {
		var errorMessages []string
		for field, message := range v.Errors {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", field, message))
		}
		return contracts.NewValidationError(strings.Join(errorMessages, "; "))
	}

	return nil
}

var sampleRequestPasswordResetRequest = &RequestPasswordResetRequest{
	Identifier: apiresource.SampleUserEmail,
}

func (*RequestPasswordResetRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRequestPasswordResetRequest)
}

const requestPasswordResetEndpointDescription = `This endpoint is utilized to request a password reset for a user. An email will be sent to the user 
with a link to reset their password. Learn more about authentication and authorization in our 
[documentation](https://docs.augno.com/guides/authentication).
`

type RequestPasswordResetEndpoint struct {
	apiendpoint.APIEndpoint[*RequestPasswordResetRequest, *apiresource.EmptyResource]

	group    *apiendpoint.APIEndpointGroup
	service  AuthCtrl
	platform constants.PlatformMode
	bindOnce sync.Once
	handler  http.HandlerFunc
}

func (e *RequestPasswordResetEndpoint) Materialize() apiendpoint.APIEndpointer {
	e.APIEndpoint = apiendpoint.APIEndpoint[*RequestPasswordResetRequest, *apiresource.EmptyResource]{
		Title:             "Request Password Reset",
		Description:       requestPasswordResetEndpointDescription,
		Method:            http.MethodPost,
		Route:             "/v1/auth/passwords/actions/request-reset",
		ContentType:       "application/json",
		Request:           &RequestPasswordResetRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusAccepted,
		IsPublic:          false,
		Handler: func(ctrl any) apiendpoint.HandlerFunc[
			*RequestPasswordResetRequest, *apiresource.EmptyResource,
		] {
			return apiendpoint.HandlerFunc[
				*RequestPasswordResetRequest, *apiresource.EmptyResource,
			](func(ctx context.Context, req *RequestPasswordResetRequest) (*apiresource.EmptyResource, *contracts.APIError) {
				return ctrl.(AuthCtrl).RequestPasswordReset(ctx, req)
			})
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
	return e
}

func (e *RequestPasswordResetEndpoint) GetHandler() http.HandlerFunc {
	e.bindOnce.Do(func() {
		be := apiendpoint.Bind(e.APIEndpoint, e.service)
		e.handler = httptransport.ConvertToHTTPHandler(be)
	})
	return e.handler
}

func (e *RequestPasswordResetEndpoint) WithGroup(g *apiendpoint.APIEndpointGroup, service AuthCtrl, platform constants.PlatformMode) *RequestPasswordResetEndpoint {
	e.group = g
	e.service = service
	e.platform = platform
	return e
}
