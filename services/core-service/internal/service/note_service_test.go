package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/actor"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
)

const noteTestAccount = "ac_notes"

func noteCallerCtx(userID string, admin bool) context.Context {
	acct := noteTestAccount
	role := string(constants.RoleTypeSalesRep)
	if admin {
		role = string(constants.RoleTypeAdmin)
	}
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: acct},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           userID,
			AccountID:    &acct,
			RoleType:     &role,
			Permissions:  map[string]bool{"notes:read": true, "notes:create": true, "notes:update": true, "notes:delete": true},
		},
	})
}

func noteBy(a actor.Actor) *domain.Note {
	return &domain.Note{ID: "nt_1", AccountID: noteTestAccount, Body: "b", Type: constants.NoteTypeComment, Importance: constants.NoteImportanceNormal, Actor: a}
}

func TestCanEditNote(t *testing.T) {
	t.Parallel()
	author := noteCallerCtx("us_author", false)
	other := noteCallerCtx("us_other", false)
	admin := noteCallerCtx("us_admin", true)
	identity := func(ctx context.Context) *types.Identity {
		i, _ := appctx.Identity[*types.Identity](ctx)
		return i
	}

	byAuthor := noteBy(actor.New(constants.ActorTypeUser, "us_author"))
	assert.True(t, canEditNote(identity(author), byAuthor), "the author edits their note")
	assert.False(t, canEditNote(identity(other), byAuthor), "someone else can't")
	assert.False(t, canEditNote(identity(admin), byAuthor), "an admin can't edit another person's words")

	bySystem := noteBy(actor.System())
	assert.True(t, canEditNote(identity(admin), bySystem), "admins look after the notes OpenMRP wrote")
	assert.False(t, canEditNote(identity(author), bySystem))

	byKey := noteBy(actor.New(constants.ActorTypeAPIKey, "us_author"))
	assert.False(t, canEditNote(identity(author), byKey), "the same ID under another actor type is someone else")
}

func TestApplyNoteUpdate(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	expires := now.Add(24 * time.Hour)
	recordType := constants.RecordTypeCustomer
	recordID := "ac_c"
	n := noteBy(actor.New(constants.ActorTypeUser, "us_a"))
	n.ExpiresAt, n.RecordType, n.RecordID = &expires, &recordType, &recordID

	same := applyNoteUpdate(n, domain.UpdateNoteParams{Body: new("b")}, now)
	assert.Nil(t, same.EditedAt, "sending the same body is no edit")

	importance := constants.NoteImportanceHigh
	next := applyNoteUpdate(n, domain.UpdateNoteParams{Body: new("changed"), Importance: &importance, ClearExpiresAt: true, ClearRecord: true}, now)
	assert.Equal(t, "changed", next.Body)
	require.NotNil(t, next.EditedAt)
	assert.Equal(t, now, *next.EditedAt)
	assert.Equal(t, constants.NoteImportanceHigh, next.Importance)
	assert.Nil(t, next.ExpiresAt)
	assert.Nil(t, next.RecordID)
	assert.Nil(t, next.RecordType)
	assert.Equal(t, "b", n.Body, "the loaded note is left as it was, for the audit diff")
}

func TestNoteStatus(t *testing.T) {
	t.Parallel()
	now := time.Now()
	past, future := now.Add(-time.Minute), now.Add(time.Minute)
	n := &domain.Note{}
	assert.Equal(t, constants.NoteStatusActive, n.Status(now))
	n.ExpiresAt = &future
	assert.Equal(t, constants.NoteStatusActive, n.Status(now))
	n.ExpiresAt = &past
	assert.Equal(t, constants.NoteStatusExpired, n.Status(now))
}

func newNoteSvcForTest(ctrl *gomock.Controller) (*noteSvcImpl, *factorymock.MockRepoFactory, *repositorymock.MockNoteRepo) {
	repos := factorymock.NewMockRepoFactory(ctrl)
	noteRepo := repositorymock.NewMockNoteRepo(ctrl)
	repos.EXPECT().NewNoteRepo().Return(noteRepo).AnyTimes()
	svc := &noteSvcImpl{repos: repos, mediatorFactory: factorymock.NewMockMediatorFactory(ctrl), txManager: &stubTxManager{factory: repos}}
	return svc, repos, noteRepo
}

func TestDeleteNote_OnlyAuthorOrAdmin(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc, repos, noteRepo := newNoteSvcForTest(ctrl)
	noteRepo.EXPECT().Get(gomock.Any(), noteTestAccount, "nt_1").Return(noteBy(actor.New(constants.ActorTypeUser, "us_author")), nil).AnyTimes()

	apiErr := svc.DeleteNote(noteCallerCtx("us_other", false), "nt_1")
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.CodeInsufficientPermissions, apiErr.Code)

	deleted := repositorymock.NewMockDeletedRecordRepo(ctrl)
	repos.EXPECT().NewDeletedRecordRepo().Return(deleted).Times(1)
	repos.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).Times(1)
	deleted.EXPECT().CreateInAccount(gomock.Any(), constants.DeletedRecordResourceTypeNote, "nt_1", noteTestAccount, gomock.Any()).Return(nil)
	noteRepo.EXPECT().Delete(gomock.Any(), noteTestAccount, "nt_1").Return(nil)
	assert.Nil(t, svc.DeleteNote(noteCallerCtx("us_admin", true), "nt_1"), "an admin deletes anyone's note")
}

func TestDeleteNote_DeletedNoteIsGone(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc, repos, noteRepo := newNoteSvcForTest(ctrl)
	noteRepo.EXPECT().Get(gomock.Any(), noteTestAccount, "nt_1").Return(nil, apierror.NewNotFoundError("no"))
	deleted := repositorymock.NewMockDeletedRecordRepo(ctrl)
	repos.EXPECT().NewDeletedRecordRepo().Return(deleted)
	deleted.EXPECT().ExistsInAccount(gomock.Any(), constants.DeletedRecordResourceTypeNote, "nt_1", noteTestAccount).Return(true, nil)

	apiErr := svc.DeleteNote(noteCallerCtx("us_author", false), "nt_1")
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.CodeResourceGone, apiErr.Code)
}

func TestImportNotes_RejectsIncompleteNotes(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc, _, _ := newNoteSvcForTest(ctrl)
	_, apiErr := svc.ImportNotes(context.Background(), []domain.ImportNote{{ID: "nt_1", AccountID: "ac_1", Type: "memo", Importance: constants.NoteImportanceLow, Actor: actor.System()}})
	assert.NotNil(t, apiErr, "an unknown type never reaches the table")
}
