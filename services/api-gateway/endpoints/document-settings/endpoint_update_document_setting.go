package documentsettingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
	"github.com/open-mrp/api/shared/validate"
)

func init() {
	validate.RegisterWrappedTypes(field.Optional[DocumentControlInput]{})
}

// Request to partially update the account's settings for one document type.
type UpdateDocumentSettingRequest struct {
	// The document type to update settings for.
	DocumentType constants.DocumentType `path:"document_type" validate:"required"`
	// Changes to the document-control block. Fields left out keep their values; pass null to remove one.
	DocumentControl field.Optional[DocumentControlInput] `json:"document_control,omitzero"`
	// Free text printed at the foot of the document. Pass null to remove it.
	FooterText field.Clearable[string] `json:"footer_text,omitzero" validate:"omitempty,max=2000"`
}

// Changes to a document-control block.
type DocumentControlInput struct {
	// The person or role accountable for the process the document records.
	ProcessOwner field.Clearable[string] `json:"process_owner,omitzero" validate:"omitempty,max=255"`
	// The document's controlled form number.
	DocumentNumber field.Clearable[string] `json:"document_number,omitzero" validate:"omitempty,max=255"`
	// The document's current revision, printed as given.
	Revision field.Clearable[string] `json:"revision,omitzero" validate:"omitempty,max=255"`
}

var sampleUpdateDocumentSettingRequest = &UpdateDocumentSettingRequest{
	DocumentControl: field.Some(DocumentControlInput{
		ProcessOwner:   field.Set("Quality Manager"),
		DocumentNumber: field.Set("FRM-QUAL-001"),
		Revision:       field.Set("Rev. C, 2026-01-15"),
	}),
}

func (*UpdateDocumentSettingRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateDocumentSettingRequest)
}

// Partially updates the account's settings for one document type.
//
// Only the fields provided are changed. The first update to a type saves it, giving it an `id`.
type UpdateDocumentSettingEndpoint struct{}

func (e *UpdateDocumentSettingEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateDocumentSettingRequest, *apiresource.DocumentSetting] {
	return (&apiendpoint.APIEndpoint[*UpdateDocumentSettingRequest, *apiresource.DocumentSetting]{
		Title:               "Update Document Setting",
		Method:              http.MethodPatch,
		Route:               "/v1/identity/document-settings/{document_type}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeDocumentSetting,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateDocumentSettingRequest) (*apiresource.DocumentSetting, *apierror.APIError) {
			return svc.(DocumentSettingSvc).UpdateDocumentSetting
		},
	})
}
