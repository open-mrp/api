package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleDocumentSettingID = "dosd_ayuuksycpo15"

// Your account's customizations to one kind of generated document, applied wherever that document is printed, exported as a PDF or emailed.
//
// Every document type has a setting. One you have never saved reads back with a null `id` and null fields, and the document is generated with its defaults.
type DocumentSetting struct {
	// Document setting ID. Null until the setting is first saved.
	ID *string `json:"id"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=document_setting"`
	// The kind of document these settings apply to.
	DocumentType constants.DocumentType `json:"document_type" validate:"required"`
	// The document-control block that identifies the document as a controlled form.
	DocumentControl DocumentControl `json:"document_control" validate:"required"`
	// Free text printed at the foot of the document.
	FooterText *string `json:"footer_text"`
	// Creation timestamp. Null until the setting is first saved.
	CreatedAt *time.Time `json:"created_at"`
	// Last updated timestamp. Null until the setting is first saved.
	UpdatedAt *time.Time `json:"updated_at"`
}

var SampleDocumentSetting = &DocumentSetting{
	ID:              new(SampleDocumentSettingID),
	Object:          constants.ObjectTypeDocumentSetting,
	DocumentType:    constants.DocumentTypeBatchTraveler,
	DocumentControl: *SampleDocumentControl,
	FooterText:      new("Retain with the batch record for seven years."),
	CreatedAt:       new(timeutil.TimestampToTime(sampleCreatedAtTimestamp)),
	UpdatedAt:       new(timeutil.TimestampToTime(sampleUpdatedAtTimestamp)),
}

func (*DocumentSetting) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleDocumentSetting)
}

// The document-control block printed on a document to identify it as a controlled form, as quality systems such as ISO 9001 require. Nothing is printed while every field is null.
type DocumentControl struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=document_control"`
	// The person or role accountable for the process the document records.
	ProcessOwner *string `json:"process_owner"`
	// The document's controlled form number.
	DocumentNumber *string `json:"document_number"`
	// The document's current revision, printed as given.
	Revision *string `json:"revision"`
}

var SampleDocumentControl = &DocumentControl{
	Object:         constants.ObjectTypeDocumentControl,
	ProcessOwner:   new("Quality Manager"),
	DocumentNumber: new("FRM-QUAL-001"),
	Revision:       new("Rev. C, 2026-01-15"),
}

func (*DocumentControl) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleDocumentControl)
}
