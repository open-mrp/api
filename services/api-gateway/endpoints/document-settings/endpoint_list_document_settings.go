package documentsettingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list the account's document settings.
type ListDocumentSettingsRequest struct{}

// Returns the account's settings for every document type.
//
// Every type is returned, in a fixed order, whether or not it has been saved; an unsaved type reads back with a null `id` and null fields.
type ListDocumentSettingsEndpoint struct{}

func (e *ListDocumentSettingsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListDocumentSettingsRequest, *apiresource.List[apiresource.DocumentSetting]] {
	return (&apiendpoint.APIEndpoint[*ListDocumentSettingsRequest, *apiresource.List[apiresource.DocumentSetting]]{
		Title:               "List Document Settings",
		Method:              http.MethodGet,
		Route:               "/v1/identity/document-settings",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeDocumentSetting,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListDocumentSettingsRequest) (*apiresource.List[apiresource.DocumentSetting], *apierror.APIError) {
			return svc.(DocumentSettingSvc).ListDocumentSettings
		},
	})
}
