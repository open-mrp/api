package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/actor"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idemresponse"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/tracing"
)

var noteSvcTracer = tracing.GetTracer("core-service.note_service")

// maxRecallNotes bounds what an agent reads before it works.
const maxRecallNotes = 100

type noteSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type NoteSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *NoteSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("note service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("note service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("note service: tx manager is required")
	}
	return nil
}

func NewNoteSvc(config *NoteSvcConfig) domain.NoteSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &noteSvcImpl{repos: config.Repos, mediatorFactory: config.MediatorFactory, txManager: config.TxManager}
}

func (s *noteSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *noteSvcImpl) withTx(ctx context.Context, fn func(context.Context, *noteSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		return fn(txCtx, &noteSvcImpl{repos: f, mediatorFactory: s.mediatorFactory, txManager: s.txManager})
	})
}

// noteIdentity is the caller, who must be a member of the account the notes belong to. Portal users never reach notes.
func noteIdentity(ctx context.Context, read bool) (*types.Identity, *apierror.APIError) {
	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}
	check := identity.CheckIsInternalActor
	if read {
		check = identity.CheckIsInternalActorForRead
	}
	if apiErr := check(); apiErr != nil {
		return nil, apiErr
	}
	return identity, nil
}

func validateNoteRecord(r *domain.NoteRecord) *apierror.APIError {
	if r == nil {
		return nil
	}
	if !r.Type.IsValid() {
		return apierror.NewFieldError("record.type", apierror.CodeInvalidFormat, "Unknown record type.")
	}
	if strings.TrimSpace(r.ID) == "" {
		return apierror.NewFieldError("record.id", apierror.CodeInvalidFormat, "A record needs an ID.")
	}
	return nil
}

func validateNoteBody(body string) *apierror.APIError {
	if strings.TrimSpace(body) == "" {
		return apierror.NewFieldError("body", apierror.CodeInvalidFormat, "A note needs a body.")
	}
	return nil
}

// notFoundOrGone is a missing note's error: 410 when it was deleted, else 404.
func (s *noteSvcImpl) notFoundOrGone(ctx context.Context, accountID, noteID string) *apierror.APIError {
	gone, apiErr := s.repos.NewDeletedRecordRepo().ExistsInAccount(ctx, constants.DeletedRecordResourceTypeNote, noteID, accountID)
	if apiErr != nil {
		return apiErr
	}
	if gone {
		return apierror.NewGoneError("This note has been deleted.")
	}
	return apierror.NewNotFoundError("Note not found.")
}

func (s *noteSvcImpl) getNote(ctx context.Context, accountID, noteID string) (*domain.Note, *apierror.APIError) {
	n, apiErr := s.repos.NewNoteRepo().Get(ctx, accountID, noteID)
	if apiErr != nil {
		if apiErr.Code == apierror.CodeResourceNotFound {
			return nil, s.notFoundOrGone(ctx, accountID, noteID)
		}
		return nil, apiErr
	}
	return n, nil
}

func (s *noteSvcImpl) ListNotes(ctx context.Context, params domain.ListNotesParams) (*domain.ListNotesResult, *apierror.APIError) {
	ctx, span := noteSvcTracer.Start(ctx, "service.note.list")
	defer span.End()

	identity, apiErr := noteIdentity(ctx, true)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainNotes, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.AccountID = identity.Target.AccountID
	result, apiErr := s.repos.NewNoteRepo().List(ctx, params)
	return result, tracing.Trace(span, apiErr)
}

func (s *noteSvcImpl) GetNote(ctx context.Context, noteID string) (*domain.Note, *apierror.APIError) {
	ctx, span := noteSvcTracer.Start(ctx, "service.note.get")
	defer span.End()

	identity, apiErr := noteIdentity(ctx, true)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainNotes, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	n, apiErr := s.getNote(ctx, identity.Target.AccountID, noteID)
	return n, tracing.Trace(span, apiErr)
}

func (s *noteSvcImpl) BatchGetNotesByIDs(ctx context.Context, ids []string) ([]*domain.Note, *apierror.APIError) {
	ctx, span := noteSvcTracer.Start(ctx, "service.note.batch_get_by_ids")
	defer span.End()

	identity, apiErr := noteIdentity(ctx, true)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainNotes, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	notes, apiErr := s.repos.NewNoteRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
	return notes, tracing.Trace(span, apiErr)
}

// CreateNote writes a note attributed to the caller.
//
// 1. Check the caller is a member of the account with notes:create.
// 2. Validate the record reference and body.
// 3. Idempotently, within a transaction: insert the note, publish the audit event and cache the response.
func (s *noteSvcImpl) CreateNote(ctx context.Context, params domain.CreateNoteParams) (*domain.Note, *apierror.APIError) {
	ctx, span := noteSvcTracer.Start(ctx, "service.note.create")
	defer span.End()

	identity, apiErr := noteIdentity(ctx, false)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainNotes, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	author, ok := actor.FromIdentity(identity)
	if !ok {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("Notes are written by a user, API key or agent."))
	}
	if apiErr := validateNoteRecord(params.Record); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := validateNoteBody(params.Body); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := requireMentionable(ctx, s.repos, identity.Target.AccountID, params.Mentions); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	note := &domain.Note{
		AccountID:  identity.Target.AccountID,
		Body:       params.Body,
		Type:       constants.NoteTypeComment,
		Importance: constants.NoteImportanceNormal,
		ExpiresAt:  params.ExpiresAt,
		Actor:      author,
	}
	if params.Type != nil {
		note.Type = *params.Type
	}
	if params.Importance != nil {
		note.Importance = *params.Importance
	}
	if params.Record != nil {
		t, rid := params.Record.Type, params.Record.ID
		note.RecordType, note.RecordID = &t, &rid
	}

	noteID, apiErr := id.GenID(id.NoteIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	note.ID = noteID

	meds := s.mediators()
	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idemresponse.UnmarshalCachedResponse[domain.Note](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Note
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *noteSvcImpl) *apierror.APIError {
			repo := txSvc.repos.NewNoteRepo()
			if apiErr := repo.Create(txCtx, note); apiErr != nil {
				return apiErr
			}
			created, apiErr := repo.Get(txCtx, note.AccountID, note.ID)
			if apiErr != nil {
				return apiErr
			}
			result = created
			if apiErr := notifyNoteMentions(txCtx, txSvc.repos, identity, created, params.Mentions); apiErr != nil {
				return apiErr
			}
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeNote,
				ResourceID:   created.ID,
				Changes:      audit.ComputeChanges(nil, created),
			}); apiErr != nil {
				return apiErr
			}
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr))
		}
		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// canEditNote is whether the caller may change the note: its author, or an admin when OpenMRP wrote it (notes copied
// from earlier fields and memories have no other author).
func canEditNote(identity *types.Identity, n *domain.Note) bool {
	if n.Actor.IsSystem() {
		return identity.IsAdmin()
	}
	caller, ok := actor.FromIdentity(identity)
	return ok && caller.Type == n.Actor.Type && caller.ID == n.Actor.ID
}

// UpdateNote changes the fields given. Changing the body stamps edited_at.
//
// 1. Check the caller is a member of the account with notes:update.
// 2. Idempotently, within a transaction: load the note, check the caller wrote it, apply the change, publish the audit
// event and cache the response.
func (s *noteSvcImpl) UpdateNote(ctx context.Context, params domain.UpdateNoteParams) (*domain.Note, *apierror.APIError) {
	ctx, span := noteSvcTracer.Start(ctx, "service.note.update")
	defer span.End()

	identity, apiErr := noteIdentity(ctx, false)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainNotes, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if params.Body != nil {
		if apiErr := validateNoteBody(*params.Body); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	if apiErr := validateNoteRecord(params.Record); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID
	if apiErr := requireMentionable(ctx, s.repos, accountID, params.Mentions); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()
	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idemresponse.UnmarshalCachedResponse[domain.Note](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Note
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *noteSvcImpl) *apierror.APIError {
			existing, apiErr := txSvc.getNote(txCtx, accountID, params.NoteID)
			if apiErr != nil {
				return apiErr
			}
			if !canEditNote(identity, existing) {
				return apierror.NewAuthorizationError("Only the note's author can edit it.")
			}
			old := *existing
			next := applyNoteUpdate(existing, params, time.Now().UTC())

			repo := txSvc.repos.NewNoteRepo()
			if apiErr := repo.Update(txCtx, next); apiErr != nil {
				return apiErr
			}
			updated, apiErr := repo.Get(txCtx, accountID, params.NoteID)
			if apiErr != nil {
				return apiErr
			}
			result = updated
			if apiErr := notifyNoteMentions(txCtx, txSvc.repos, identity, updated, params.Mentions); apiErr != nil {
				return apiErr
			}
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeNote,
				ResourceID:   updated.ID,
				Changes:      audit.ComputeChanges(&old, updated),
			}); apiErr != nil {
				return apiErr
			}
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr))
		}
		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// applyNoteUpdate is n with params applied. edited_at moves only when the body actually changes.
func applyNoteUpdate(n *domain.Note, params domain.UpdateNoteParams, now time.Time) *domain.Note {
	next := *n
	if params.Body != nil && *params.Body != n.Body {
		next.Body = *params.Body
		next.EditedAt = &now
	}
	if params.Type != nil {
		next.Type = *params.Type
	}
	if params.Importance != nil {
		next.Importance = *params.Importance
	}
	switch {
	case params.ClearExpiresAt:
		next.ExpiresAt = nil
	case params.ExpiresAt != nil:
		next.ExpiresAt = params.ExpiresAt
	}
	switch {
	case params.ClearRecord:
		next.RecordType, next.RecordID = nil, nil
	case params.Record != nil:
		t, rid := params.Record.Type, params.Record.ID
		next.RecordType, next.RecordID = &t, &rid
	}
	return &next
}

// DeleteNote removes a note, leaving a tombstone so a repeated delete reads 410.
//
// 1. Check the caller is a member of the account with notes:delete.
// 2. Within a transaction: load the note, check the caller wrote it or is an admin, record the tombstone, delete it and
// publish the audit event.
func (s *noteSvcImpl) DeleteNote(ctx context.Context, noteID string) *apierror.APIError {
	ctx, span := noteSvcTracer.Start(ctx, "service.note.delete")
	defer span.End()

	identity, apiErr := noteIdentity(ctx, false)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainNotes, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *noteSvcImpl) *apierror.APIError {
		existing, apiErr := txSvc.getNote(txCtx, accountID, noteID)
		if apiErr != nil {
			return apiErr
		}
		if !identity.IsAdmin() && !canEditNote(identity, existing) {
			return apierror.NewAuthorizationError("Only the note's author or an admin can delete it.")
		}
		if apiErr := txSvc.repos.NewDeletedRecordRepo().CreateInAccount(txCtx, constants.DeletedRecordResourceTypeNote, existing.ID, accountID, existing); apiErr != nil {
			return apiErr
		}
		if apiErr := txSvc.repos.NewNoteRepo().Delete(txCtx, accountID, noteID); apiErr != nil {
			return apiErr
		}
		return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeNote,
			ResourceID:   noteID,
			Changes:      audit.ComputeChanges(existing, (*domain.Note)(nil)),
		})
	})
	return tracing.Trace(span, apiErr)
}

func (s *noteSvcImpl) ImportNotes(ctx context.Context, notes []domain.ImportNote) (int64, *apierror.APIError) {
	ctx, span := noteSvcTracer.Start(ctx, "service.note.import")
	defer span.End()

	for _, n := range notes {
		if err := n.Actor.Validate(); err != nil {
			return 0, tracing.Trace(span, apierror.NewInvariantViolationError(fmt.Sprintf("note %s: %v", n.ID, err)))
		}
		if n.ID == "" || n.AccountID == "" || !n.Type.IsValid() || !n.Importance.IsValid() {
			return 0, tracing.Trace(span, apierror.NewInvariantViolationError("note import: incomplete note "+n.ID))
		}
		if n.Record != nil && !n.Record.Type.IsValid() {
			return 0, tracing.Trace(span, apierror.NewInvariantViolationError("note import: unknown record type on "+n.ID))
		}
	}
	inserted, apiErr := s.repos.NewNoteRepo().Import(ctx, notes)
	return inserted, tracing.Trace(span, apiErr)
}

func (s *noteSvcImpl) RecallNotes(ctx context.Context, params domain.RecallNotesParams) ([]*domain.Note, *apierror.APIError) {
	ctx, span := noteSvcTracer.Start(ctx, "service.note.recall")
	defer span.End()

	if params.AccountID == "" {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("note recall: account is required"))
	}
	if params.Limit <= 0 || params.Limit > maxRecallNotes {
		params.Limit = maxRecallNotes
	}
	notes, apiErr := s.repos.NewNoteRepo().Recall(ctx, params)
	return notes, tracing.Trace(span, apiErr)
}
