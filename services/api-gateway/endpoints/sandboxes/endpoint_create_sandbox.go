package sandboxep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to create a sandbox.
type CreateSandboxRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Controls how the sandbox is initialized.
	//
	// - `blank`: starts empty, with no pre-populated data.
	// - `seeded`: starts with sample data, populated asynchronously after the sandbox is created.
	Mode field.Optional[constants.SandboxMode] `json:"mode,omitzero" default:"blank"`
}

var sampleSandboxMode = constants.SandboxModeBlank

var sampleCreateSandboxRequest = &CreateSandboxRequest{
	Name: "Integration Testing",
	Mode: field.Some(sampleSandboxMode),
}

func (*CreateSandboxRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateSandboxRequest)
}

// Creates a sandbox account owned by your production account.
//
// When `mode` is `seeded`, sample data is populated asynchronously and may not be available immediately after the sandbox is created. Sandboxes cannot be created while acting in a sandbox.
type CreateSandboxEndpoint struct{}

func (e *CreateSandboxEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSandboxRequest, *apiresource.Sandbox] {
	return (&apiendpoint.APIEndpoint[*CreateSandboxRequest, *apiresource.Sandbox]{
		Title:             "Create Sandbox",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/core/sandboxes",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSandbox,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSandbox,
			Fields:     []string{"owner_account"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSandboxRequest) (*apiresource.Sandbox, *apierror.APIError) {
			return svc.(SandboxSvc).CreateSandbox
		},
		LocationFunc: func(resp *apiresource.Sandbox) string {
			return "/v1/core/sandboxes/" + resp.ID
		},
	})
}
