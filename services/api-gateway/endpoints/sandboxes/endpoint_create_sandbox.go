package sandboxep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

const createSandboxEndpointDescription string = `This endpoint creates a new sandbox account for the target account.
Enforces a per-account sandbox limit. Requires admin permissions.`

// The request to create a sandbox.
type CreateSandboxRequest struct {
	// The display name for the sandbox.
	Name string `json:"name" validate:"required"`
	// Controls whether the sandbox is blank or seeded with tutorial data.
	Mode constants.SandboxMode `json:"mode,omitempty" validate:"omitempty"`
}

var sampleCreateSandboxRequest = &CreateSandboxRequest{
	Name: "Integration Testing",
	Mode: "blank",
}

func (*CreateSandboxRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateSandboxRequest)
}

type CreateSandboxEndpoint struct{}

func (e *CreateSandboxEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSandboxRequest, *apiresource.Sandbox] {
	return &apiendpoint.APIEndpoint[*CreateSandboxRequest, *apiresource.Sandbox]{
		Title:             "Create Sandbox",
		Description:       createSandboxEndpointDescription,
		Method:            http.MethodPost,
		Route:             "/v1/core/sandboxes",
		Request:           &CreateSandboxRequest{},
		Response:          apiresource.SampleSandbox,
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		IncludeConfig: &apiendpoint.IncludeConfig{
			Fields: []apiendpoint.IncludeField{
				{Key: "owner_account", ObjectType: constants.ObjectTypeAccount, JSONPaths: []string{"owner_account"}},
			},
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSandboxRequest) (*apiresource.Sandbox, *apierror.APIError) {
			return svc.(SandboxSvc).CreateSandbox
		},
	}
}
