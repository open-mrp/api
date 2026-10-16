package roleep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to preview a role's permissions.
type PreviewRoleRequest struct {
	// A template to start from; its access and grants apply unless you set them.
	Template field.Optional[constants.RoleTemplate] `json:"template,omitzero"`
	// Access levels per area. Areas left out take the template's level, or `none`.
	Access *RoleAccessInput `json:"access,omitzero"`
	// Permission codes to hold beyond the access levels. Replaces the template's grants when sent.
	Grants field.Optional[[]string] `json:"grants,omitzero"`
	// Permission codes the access levels include that the role shouldn't hold.
	Revokes []string `json:"revokes,omitzero"`
}

var samplePreviewRoleRequest = &PreviewRoleRequest{
	Template: field.Some(constants.RoleTemplateSales),
	Grants:   field.Some([]string{"invoices:read"}),
}

func (*PreviewRoleRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(samplePreviewRoleRequest)
}

// Previews the permissions a role would hold.
//
// Takes the same `template`, `access`, `grants` and `revokes` as creating a role and returns every permission they add up to, without saving anything.
type PreviewRoleEndpoint struct{}

func (e *PreviewRoleEndpoint) Materialize() *apiendpoint.APIEndpoint[*PreviewRoleRequest, *apiresource.List[apiresource.Permission]] {
	return (&apiendpoint.APIEndpoint[*PreviewRoleRequest, *apiresource.List[apiresource.Permission]]{
		Title:             "Preview Role",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/identity/roles/actions/preview",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ReadOnly:          true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainRoles, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *PreviewRoleRequest) (*apiresource.List[apiresource.Permission], *apierror.APIError) {
			return svc.(RoleSvc).PreviewRole
		},
		ObjectType: constants.ObjectTypePermission,
	})
}
