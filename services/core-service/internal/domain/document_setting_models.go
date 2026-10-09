package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/field"
)

// DocumentSetting is an account's customizations to one document type. An account that has never saved the
// type reads back one with an empty ID and every field nil.
type DocumentSetting struct {
	ID             string
	AccountID      string
	DocumentType   constants.DocumentType
	ProcessOwner   *string `audit:"document_control.process_owner"`
	DocumentNumber *string `audit:"document_control.document_number"`
	Revision       *string `audit:"document_control.revision"`
	FooterText     *string `audit:"footer_text"`
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
}

// UpdateDocumentSettingParams holds the fields of a document setting update. A cleared field is removed.
type UpdateDocumentSettingParams struct {
	DocumentType   constants.DocumentType
	ProcessOwner   field.Clearable[string]
	DocumentNumber field.Clearable[string]
	Revision       field.Clearable[string]
	FooterText     field.Clearable[string]
}

// After is the setting with this update's fields applied to before.
func (p *UpdateDocumentSettingParams) After(before DocumentSetting) DocumentSetting {
	after := before
	after.ProcessOwner = p.ProcessOwner.StringPtrAfterBackfill(before.ProcessOwner)
	after.DocumentNumber = p.DocumentNumber.StringPtrAfterBackfill(before.DocumentNumber)
	after.Revision = p.Revision.StringPtrAfterBackfill(before.Revision)
	after.FooterText = p.FooterText.StringPtrAfterBackfill(before.FooterText)
	return after
}
