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

// Request to retrieve the account's settings for one document type.
type RetrieveDocumentSettingRequest struct {
	// The document type to retrieve settings for.
	DocumentType constants.DocumentType `path:"document_type" validate:"required"`
}

// Retrieves the account's settings for one document type.
//
// A type that has never been saved reads back with a null `id` and null fields rather than a not-found error.
type RetrieveDocumentSettingEndpoint struct{}

func (e *RetrieveDocumentSettingEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveDocumentSettingRequest, *apiresource.DocumentSetting] {
	return (&apiendpoint.APIEndpoint[*RetrieveDocumentSettingRequest, *apiresource.DocumentSetting]{
		Title:               "Retrieve Document Setting",
		Method:              http.MethodGet,
		Route:               "/v1/identity/document-settings/{document_type}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeDocumentSetting,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveDocumentSettingRequest) (*apiresource.DocumentSetting, *apierror.APIError) {
			return svc.(DocumentSettingSvc).GetDocumentSetting
		},
	})
}
