package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type documentSettingSvcSetup struct {
	svc         domain.DocumentSettingSvc
	settings    *repositorymock.MockDocumentSettingRepo
	idempotency *mediatormock.MockIdempotencyMed
	outbox      *recordingOutboxRepo
}

func newDocumentSettingSvcSetup(t *testing.T) *documentSettingSvcSetup {
	ctrl := gomock.NewController(t)
	s := &documentSettingSvcSetup{
		settings:    repositorymock.NewMockDocumentSettingRepo(ctrl),
		idempotency: mediatormock.NewMockIdempotencyMed(ctrl),
		outbox:      &recordingOutboxRepo{},
	}
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewDocumentSettingRepo().Return(s.settings).AnyTimes()
	repos.EXPECT().NewOutboxRepo().Return(s.outbox).AnyTimes()
	mediators := factorymock.NewMockMediatorFactory(ctrl)
	mediators.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: s.idempotency}).AnyTimes()

	s.svc = NewDocumentSettingSvc(&DocumentSettingSvcConfig{
		Repos:           repos,
		MediatorFactory: mediators,
		TxManager:       &stubTxManager{factory: repos},
	})
	return s
}

func (s *documentSettingSvcSetup) expectWrite() {
	s.idempotency.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{TypeID: "idk_doc", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil)
	s.idempotency.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_doc", gomock.Any()).Return(nil).AnyTimes()
	s.idempotency.EXPECT().CacheErrorResponse(gomock.Any(), "idk_doc", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError { return apiErr }).AnyTimes()
}

type documentSettingAuditEvent struct {
	Action  constants.AuditAction `json:"action"`
	Changes []audit.FieldChange   `json:"changes"`
}

func (s *documentSettingSvcSetup) auditEvents(t *testing.T) []documentSettingAuditEvent {
	t.Helper()
	events := make([]documentSettingAuditEvent, len(s.outbox.messages))
	for i, msg := range s.outbox.messages {
		require.NoError(t, json.Unmarshal(msg.Payload.Data, &events[i]))
	}
	return events
}

func changedFields(changes []audit.FieldChange) []string {
	fields := make([]string, len(changes))
	for i, c := range changes {
		fields[i] = c.Field
	}
	return fields
}

func savedDocumentSetting(documentType constants.DocumentType, mutate func(*domain.DocumentSetting)) *domain.DocumentSetting {
	now := time.Now()
	s := &domain.DocumentSetting{
		ID: "dosd_1", AccountID: "ac_1", DocumentType: documentType,
		CreatedAt: &now, UpdatedAt: &now,
	}
	if mutate != nil {
		mutate(s)
	}
	return s
}

func TestDocumentSettingSvc_ListFillsUnsavedTypesInOrder(t *testing.T) {
	t.Parallel()
	s := newDocumentSettingSvcSetup(t)
	traveler := savedDocumentSetting(constants.DocumentTypeBatchTraveler, func(d *domain.DocumentSetting) {
		d.FooterText = new("Keep with the batch.")
	})
	s.settings.EXPECT().List(gomock.Any(), "ac_1").Return([]*domain.DocumentSetting{traveler}, nil)

	got, apiErr := s.svc.ListDocumentSettings(territoryInternalCtx("ac_1"))
	require.Nil(t, apiErr)

	types := constants.DocumentTypes()
	require.Len(t, got, len(types))
	for i, documentType := range types {
		assert.Equal(t, documentType, got[i].DocumentType)
		if documentType == constants.DocumentTypeBatchTraveler {
			assert.Same(t, traveler, got[i])
			continue
		}
		assert.Empty(t, got[i].ID, "%s is unsaved", documentType)
		assert.Equal(t, "ac_1", got[i].AccountID)
		assert.Nil(t, got[i].FooterText)
	}
}

func TestDocumentSettingSvc_GetUnknownTypeIsRejected(t *testing.T) {
	t.Parallel()
	s := newDocumentSettingSvcSetup(t)

	_, apiErr := s.svc.GetDocumentSetting(territoryInternalCtx("ac_1"), "letterhead")
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeValidationFailed, apiErr.Code)
	assert.Equal(t, "document_type", apiErr.Param)
}

func TestDocumentSettingSvc_GetUnsavedReturnsDefault(t *testing.T) {
	t.Parallel()
	s := newDocumentSettingSvcSetup(t)
	s.settings.EXPECT().Get(gomock.Any(), "ac_1", constants.DocumentTypeInvoice).Return(nil, nil)

	got, apiErr := s.svc.GetDocumentSetting(territoryInternalCtx("ac_1"), constants.DocumentTypeInvoice)
	require.Nil(t, apiErr)
	assert.Equal(t, &domain.DocumentSetting{AccountID: "ac_1", DocumentType: constants.DocumentTypeInvoice}, got)
}

func TestDocumentSettingSvc_FirstUpdateCreates(t *testing.T) {
	t.Parallel()
	s := newDocumentSettingSvcSetup(t)
	s.expectWrite()

	stored := savedDocumentSetting(constants.DocumentTypeBatchTraveler, func(d *domain.DocumentSetting) {
		d.ProcessOwner = new("Quality Manager")
		d.Revision = new("Rev. A")
	})
	gomock.InOrder(
		s.settings.EXPECT().Get(gomock.Any(), "ac_1", constants.DocumentTypeBatchTraveler).Return(nil, nil),
		s.settings.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, settingID string, setting domain.DocumentSetting) *apierror.APIError {
				assert.Regexp(t, `^dosd_`, settingID)
				assert.Equal(t, "ac_1", setting.AccountID)
				assert.Equal(t, constants.DocumentTypeBatchTraveler, setting.DocumentType)
				assert.Equal(t, new("Quality Manager"), setting.ProcessOwner)
				assert.Nil(t, setting.DocumentNumber)
				assert.Equal(t, new("Rev. A"), setting.Revision)
				return nil
			}),
		s.settings.EXPECT().Get(gomock.Any(), "ac_1", constants.DocumentTypeBatchTraveler).Return(stored, nil),
	)

	got, apiErr := s.svc.UpdateDocumentSetting(territoryInternalCtx("ac_1"), domain.UpdateDocumentSettingParams{
		DocumentType: constants.DocumentTypeBatchTraveler,
		ProcessOwner: field.Set("Quality Manager"),
		Revision:     field.Set("Rev. A"),
	})
	require.Nil(t, apiErr)
	assert.Same(t, stored, got)

	events := s.auditEvents(t)
	require.Len(t, events, 1)
	assert.Equal(t, constants.AuditActionCreate, events[0].Action)
	assert.Equal(t, []string{"document_control.process_owner", "document_control.revision"}, changedFields(events[0].Changes))
}

func TestDocumentSettingSvc_UpdateMergesAndClears(t *testing.T) {
	t.Parallel()
	s := newDocumentSettingSvcSetup(t)
	s.expectWrite()

	before := savedDocumentSetting(constants.DocumentTypePackList, func(d *domain.DocumentSetting) {
		d.ProcessOwner = new("Shipping Lead")
		d.DocumentNumber = new("FRM-SHIP-002")
		d.FooterText = new("Thanks!")
	})
	after := savedDocumentSetting(constants.DocumentTypePackList, func(d *domain.DocumentSetting) {
		d.DocumentNumber = new("FRM-SHIP-002")
		d.FooterText = new("Thanks!")
	})
	gomock.InOrder(
		s.settings.EXPECT().Get(gomock.Any(), "ac_1", constants.DocumentTypePackList).Return(before, nil),
		s.settings.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, setting domain.DocumentSetting) *apierror.APIError {
				assert.Nil(t, setting.ProcessOwner, "a cleared field is written as NULL")
				assert.Equal(t, new("FRM-SHIP-002"), setting.DocumentNumber, "an omitted field keeps its value")
				assert.Equal(t, new("Thanks!"), setting.FooterText)
				return nil
			}),
		s.settings.EXPECT().Get(gomock.Any(), "ac_1", constants.DocumentTypePackList).Return(after, nil),
	)

	_, apiErr := s.svc.UpdateDocumentSetting(territoryInternalCtx("ac_1"), domain.UpdateDocumentSettingParams{
		DocumentType: constants.DocumentTypePackList,
		ProcessOwner: field.Clear[string](),
	})
	require.Nil(t, apiErr)

	events := s.auditEvents(t)
	require.Len(t, events, 1)
	assert.Equal(t, constants.AuditActionUpdate, events[0].Action)
	assert.Equal(t, []string{"document_control.process_owner"}, changedFields(events[0].Changes))
}

func TestDocumentSettingSvc_UpdateUnknownTypeIsRejected(t *testing.T) {
	t.Parallel()
	s := newDocumentSettingSvcSetup(t)

	_, apiErr := s.svc.UpdateDocumentSetting(territoryInternalCtx("ac_1"), domain.UpdateDocumentSettingParams{DocumentType: "letterhead"})
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeValidationFailed, apiErr.Code)
}
