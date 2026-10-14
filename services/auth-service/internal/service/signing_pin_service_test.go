package service

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/open-mrp/api/services/auth-service/internal/domain"
	clientmock "github.com/open-mrp/api/services/auth-service/internal/domain/mock/client"
	factorymock "github.com/open-mrp/api/services/auth-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/auth-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/auth-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/services/auth-service/internal/mediator"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/crypto"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	pinTestAccountID    = "acct_pin"
	pinTestOtherAccount = "acct_other"
	pinTestCallerUserID = "usr_caller"
	pinTestSignerUserID = "usr_signer"
	pinTestSignerAUID   = "acu_signer"
	pinTestDepartmentID = "dep_finishing"
	pinTestRoleID       = "rol_supervisor"
	pinTestCorrectPin   = "2580"
	pinTestReplacement  = "8642"
	pinTestWrongPin     = "1357"
	pinTestCallerAUID   = "acu_caller"
)

// fakeSigningPinRepo mirrors the SQL semantics of the signing_pin queries in memory.
type fakeSigningPinRepo struct {
	mu   sync.Mutex
	pins map[string]domain.SigningPin
	now  func() time.Time
}

func (r *fakeSigningPinRepo) FindByUserID(_ context.Context, userID string) (*domain.SigningPin, *apierror.APIError) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.pins[userID]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Resource not found.")
	}
	return &p, nil
}

func (r *fakeSigningPinRepo) Upsert(_ context.Context, userID, pinHash string, isTemporary bool) *apierror.APIError {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pins[userID] = domain.SigningPin{UserID: userID, PinHash: pinHash, IsTemporary: isTemporary, SetAt: r.now()}
	return nil
}

func (r *fakeSigningPinRepo) Delete(_ context.Context, userID string) (bool, *apierror.APIError) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.pins[userID]
	delete(r.pins, userID)
	return ok, nil
}

func (r *fakeSigningPinRepo) RecordFailure(_ context.Context, userID, pinHash string, maxAttempts int, lockUntil time.Time) *apierror.APIError {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.pins[userID]
	if !ok || p.PinHash != pinHash {
		return nil
	}
	if p.FailedAttempts+1 >= maxAttempts {
		p.LockedUntil = &lockUntil
		p.FailedAttempts = 0
	} else {
		p.FailedAttempts++
	}
	r.pins[userID] = p
	return nil
}

func (r *fakeSigningPinRepo) ResetFailures(_ context.Context, userID string) *apierror.APIError {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.pins[userID]
	if !ok {
		return nil
	}
	p.FailedAttempts = 0
	p.LockedUntil = nil
	r.pins[userID] = p
	return nil
}

type recordingOutboxRepo struct {
	mu     sync.Mutex
	events []map[string]any
}

func (r *recordingOutboxRepo) Create(_ context.Context, input messaging.OutboxMessageInput) (int64, error) {
	var event map[string]any
	if err := json.Unmarshal(input.Payload.Data, &event); err != nil {
		return 0, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
	return int64(len(r.events)), nil
}

func (r *recordingOutboxRepo) signingPinEvents() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for _, e := range r.events {
		meta, _ := e["metadata"].(map[string]any)
		out = append(out, meta["signing_pin_event"].(string))
	}
	return out
}

type signingPinFixture struct {
	svc      domain.SigningPinSvc
	pins     *fakeSigningPinRepo
	users    *repositorymock.MockUserRepo
	core     *clientmock.MockAuthCoreClient
	outbox   *recordingOutboxRepo
	clock    time.Time
	idemKeys int
}

func newSigningPinFixture(t *testing.T) *signingPinFixture {
	t.Helper()
	ctrl := gomock.NewController(t)

	f := &signingPinFixture{
		clock:  time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC),
		users:  repositorymock.NewMockUserRepo(ctrl),
		core:   clientmock.NewMockAuthCoreClient(ctrl),
		outbox: &recordingOutboxRepo{},
	}
	now := func() time.Time { return f.clock }
	f.pins = &fakeSigningPinRepo{pins: map[string]domain.SigningPin{}, now: now}

	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewSigningPinRepo().Return(f.pins).AnyTimes()
	repos.EXPECT().NewUserRepo().Return(f.users).AnyTimes()
	repos.EXPECT().NewOutboxRepo().Return(f.outbox).AnyTimes()

	idem := mediatormock.NewMockIdempotencyMed(ctrl)
	idem.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).DoAndReturn(
		func(context.Context, *domain.RequestIdentity) (*domain.IdempotencyKey, *apierror.APIError) {
			f.idemKeys++
			return &domain.IdempotencyKey{TypeID: "sik_test", RecoveryPoint: domain.RecoveryPointStarted}, nil
		}).AnyTimes()
	idem.EXPECT().CacheSuccessResponse(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	idem.EXPECT().CacheErrorResponse(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError { return apiErr }).AnyTimes()

	f.svc = NewSigningPinSvc(&SigningPinSvcConfig{
		Repos:     repos,
		TxManager: stubTxManager{repoFactory: repos},
		MediatorFactory: staticMediatorFactory{mediators: domain.Mediators{
			Idempotency: idem,
			SigningPin:  mediator.NewSigningPinMed(&mediator.SigningPinMedConfig{Repos: repos, Now: now}),
		}},
		CoreClient: f.core,
	})
	return f
}

func (f *signingPinFixture) expectSigner(accountID string, status string) {
	name := "Pat Signer"
	dept := pinTestDepartmentID
	f.users.EXPECT().FindAccountUser(gomock.Any(), accountID, pinTestSignerAUID).Return(&domain.SigningAccountUser{
		ID:           pinTestSignerAUID,
		UserID:       pinTestSignerUserID,
		AccountID:    accountID,
		DepartmentID: &dept,
		StatusCode:   status,
		Name:         &name,
	}, nil).AnyTimes()
}

func (f *signingPinFixture) storePin(t *testing.T, pin string, temporary bool) {
	t.Helper()
	hash, err := crypto.HashBcrypt(pin)
	require.NoError(t, err)
	require.Nil(t, f.pins.Upsert(context.Background(), pinTestSignerUserID, hash, temporary))
}

func pinIdentity(roleType constants.RoleType, permissions ...types.Permission) *types.Identity {
	acct := pinTestAccountID
	rt := string(roleType)
	perms := map[string]bool{}
	for _, p := range permissions {
		perms[p.String()] = true
	}
	return &types.Identity{
		Type: types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           pinTestCallerUserID,
			AccountID:    &acct,
			RoleType:     &rt,
			Permissions:  perms,
		},
		Target: &types.IdentityTarget{AccountID: pinTestAccountID},
	}
}

func pinCtx(identity *types.Identity) context.Context {
	return appctx.WithIdentity(context.Background(), identity)
}

var teamUpdate = types.Permission{Domain: types.PermissionDomainTeamUsers, Action: types.ActionUpdate}

func requireCode(t *testing.T, apiErr *apierror.APIError, code apierror.ErrorCode) {
	t.Helper()
	require.NotNil(t, apiErr)
	assert.Equal(t, code, apiErr.Code, apiErr.Error())
}

func TestVerifySigningPin_LockoutAfterFiveWrongPins(t *testing.T) {
	t.Parallel()
	f := newSigningPinFixture(t)
	f.expectSigner(pinTestAccountID, "active")
	f.storePin(t, pinTestCorrectPin, false)
	ctx := pinCtx(pinIdentity(constants.RoleTypeScanner))

	for i := 1; i <= 4; i++ {
		_, apiErr := f.svc.VerifySigningPin(ctx, pinTestSignerAUID, pinTestWrongPin)
		requireCode(t, apiErr, apierror.ErrorCodeSigningPINInvalid)
		assert.Equal(t, i, f.pins.pins[pinTestSignerUserID].FailedAttempts)
	}

	_, apiErr := f.svc.VerifySigningPin(ctx, pinTestSignerAUID, pinTestWrongPin)
	requireCode(t, apiErr, apierror.ErrorCodeSigningPINLocked)
	stored := f.pins.pins[pinTestSignerUserID]
	require.NotNil(t, stored.LockedUntil)
	assert.Equal(t, f.clock.Add(15*time.Minute), *stored.LockedUntil)
	assert.Equal(t, 0, stored.FailedAttempts)

	_, apiErr = f.svc.VerifySigningPin(ctx, pinTestSignerAUID, pinTestCorrectPin)
	requireCode(t, apiErr, apierror.ErrorCodeSigningPINLocked)

	assert.Equal(t, []string{
		"verification_failed", "verification_failed", "verification_failed", "verification_failed",
		"verification_failed", "locked",
	}, f.outbox.signingPinEvents())

	f.clock = f.clock.Add(15*time.Minute + time.Second)
	f.core.EXPECT().GetUserAccountAccess(gomock.Any(), pinTestSignerUserID, pinTestAccountID).Return(&domain.AccountUserAccess{
		AccountUserID: pinTestSignerAUID,
		AccountID:     pinTestAccountID,
		RoleID:        new(pinTestRoleID),
		RoleName:      new("Supervisor"),
		RoleType:      new(string(constants.RoleTypeCustom)),
		Permissions:   map[string]bool{"batches:create": true, "batch_releases:create": true, "items:read": false},
	}, true, nil)

	signer, apiErr := f.svc.VerifySigningPin(ctx, pinTestSignerAUID, pinTestCorrectPin)
	require.Nil(t, apiErr)
	assert.Equal(t, &domain.VerifiedSigner{
		AccountUserID: pinTestSignerAUID,
		UserID:        pinTestSignerUserID,
		Name:          "Pat Signer",
		RoleID:        new(pinTestRoleID),
		RoleName:      new("Supervisor"),
		RoleTypeCode:  new(string(constants.RoleTypeCustom)),
		DepartmentID:  new(pinTestDepartmentID),
		Permissions:   []string{"batch_releases:create", "batches:create"},
	}, signer)
	stored = f.pins.pins[pinTestSignerUserID]
	assert.Nil(t, stored.LockedUntil)
	assert.Equal(t, 0, stored.FailedAttempts)
}

func TestVerifySigningPin_CorrectPinResetsCount(t *testing.T) {
	t.Parallel()
	f := newSigningPinFixture(t)
	f.expectSigner(pinTestAccountID, "active")
	f.storePin(t, pinTestCorrectPin, false)
	ctx := pinCtx(pinIdentity(constants.RoleTypeScanner))
	f.core.EXPECT().GetUserAccountAccess(gomock.Any(), pinTestSignerUserID, pinTestAccountID).Return(&domain.AccountUserAccess{
		RoleType:    new(string(constants.RoleTypeAdmin)),
		Permissions: map[string]bool{"batches:create": true},
	}, true, nil)

	_, apiErr := f.svc.VerifySigningPin(ctx, pinTestSignerAUID, pinTestWrongPin)
	requireCode(t, apiErr, apierror.ErrorCodeSigningPINInvalid)

	signer, apiErr := f.svc.VerifySigningPin(ctx, pinTestSignerAUID, pinTestCorrectPin)
	require.Nil(t, apiErr)
	assert.Empty(t, signer.Permissions, "admins hold every permission, so none are listed")
	assert.NotNil(t, signer.Permissions)
	assert.Equal(t, 0, f.pins.pins[pinTestSignerUserID].FailedAttempts)
}

func TestVerifySigningPin_NotSet(t *testing.T) {
	t.Parallel()
	f := newSigningPinFixture(t)
	f.expectSigner(pinTestAccountID, "active")

	_, apiErr := f.svc.VerifySigningPin(pinCtx(pinIdentity(constants.RoleTypeScanner)), pinTestSignerAUID, pinTestCorrectPin)
	requireCode(t, apiErr, apierror.ErrorCodeSigningPINNotSet)
	assert.Empty(t, f.outbox.signingPinEvents())
}

func TestVerifySigningPin_AccountScoping(t *testing.T) {
	t.Parallel()
	f := newSigningPinFixture(t)
	f.storePin(t, pinTestCorrectPin, false)
	f.users.EXPECT().FindAccountUser(gomock.Any(), pinTestAccountID, pinTestSignerAUID).
		Return(nil, apierror.NewResourceNotFoundError("Account user not found."))

	_, apiErr := f.svc.VerifySigningPin(pinCtx(pinIdentity(constants.RoleTypeScanner)), pinTestSignerAUID, pinTestCorrectPin)
	requireCode(t, apiErr, apierror.ErrorCodeResourceNotFound)
}

func TestVerifySigningPin_InactiveAccountUser(t *testing.T) {
	t.Parallel()
	f := newSigningPinFixture(t)
	f.expectSigner(pinTestAccountID, "disabled")
	f.storePin(t, pinTestCorrectPin, false)

	_, apiErr := f.svc.VerifySigningPin(pinCtx(pinIdentity(constants.RoleTypeScanner)), pinTestSignerAUID, pinTestCorrectPin)
	requireCode(t, apiErr, apierror.ErrorCodeResourceConflict)
}

func TestVerifySigningPin_RequiresInternalActorInTargetAccount(t *testing.T) {
	t.Parallel()
	f := newSigningPinFixture(t)
	identity := pinIdentity(constants.RoleTypeScanner)
	identity.Target.AccountID = pinTestOtherAccount

	_, apiErr := f.svc.VerifySigningPin(pinCtx(identity), pinTestSignerAUID, pinTestCorrectPin)
	requireCode(t, apiErr, apierror.ErrorCodeInsufficientPerms)
}

func TestTemporarySigningPin_MustBeChangedBeforeUse(t *testing.T) {
	t.Parallel()
	f := newSigningPinFixture(t)
	f.expectSigner(pinTestAccountID, "active")
	admin := pinCtx(pinIdentity(constants.RoleTypeCustom, teamUpdate))
	station := pinCtx(pinIdentity(constants.RoleTypeScanner))

	require.Nil(t, f.svc.SetTemporarySigningPin(admin, pinTestSignerAUID, pinTestCorrectPin))
	assert.True(t, f.pins.pins[pinTestSignerUserID].IsTemporary)

	_, apiErr := f.svc.VerifySigningPin(station, pinTestSignerAUID, pinTestCorrectPin)
	requireCode(t, apiErr, apierror.ErrorCodeSigningPINChangeRequired)

	apiErr = f.svc.ChangeSigningPin(station, pinTestSignerAUID, pinTestWrongPin, pinTestReplacement)
	requireCode(t, apiErr, apierror.ErrorCodeSigningPINInvalid)
	assert.Equal(t, 1, f.pins.pins[pinTestSignerUserID].FailedAttempts)

	apiErr = f.svc.ChangeSigningPin(station, pinTestSignerAUID, pinTestCorrectPin, pinTestCorrectPin)
	requireCode(t, apiErr, apierror.ErrorCodeValidationFailed)
	assert.Equal(t, "new_pin", apiErr.Param)

	apiErr = f.svc.ChangeSigningPin(station, pinTestSignerAUID, pinTestCorrectPin, "1234")
	requireCode(t, apiErr, apierror.ErrorCodeValidationFailed)
	assert.Equal(t, "new_pin", apiErr.Param)

	require.Nil(t, f.svc.ChangeSigningPin(station, pinTestSignerAUID, pinTestCorrectPin, pinTestReplacement))
	stored := f.pins.pins[pinTestSignerUserID]
	assert.False(t, stored.IsTemporary)
	assert.Equal(t, 0, stored.FailedAttempts)

	f.core.EXPECT().GetUserAccountAccess(gomock.Any(), pinTestSignerUserID, pinTestAccountID).
		Return(&domain.AccountUserAccess{Permissions: map[string]bool{}}, true, nil)
	_, apiErr = f.svc.VerifySigningPin(station, pinTestSignerAUID, pinTestReplacement)
	require.Nil(t, apiErr)

	assert.Equal(t, []string{"temporary_set", "verification_failed", "changed"}, f.outbox.signingPinEvents())
}

func TestChangeSigningPin_LockedRefusesEvenCorrectPin(t *testing.T) {
	t.Parallel()
	f := newSigningPinFixture(t)
	f.expectSigner(pinTestAccountID, "active")
	f.storePin(t, pinTestCorrectPin, false)
	station := pinCtx(pinIdentity(constants.RoleTypeScanner))

	for range 5 {
		_ = f.svc.ChangeSigningPin(station, pinTestSignerAUID, pinTestWrongPin, pinTestReplacement)
	}
	apiErr := f.svc.ChangeSigningPin(station, pinTestSignerAUID, pinTestCorrectPin, pinTestReplacement)
	requireCode(t, apiErr, apierror.ErrorCodeSigningPINLocked)
}

func TestSetTemporarySigningPin_RequiresTeamUpdate(t *testing.T) {
	t.Parallel()
	f := newSigningPinFixture(t)

	apiErr := f.svc.SetTemporarySigningPin(pinCtx(pinIdentity(constants.RoleTypeCustom)), pinTestSignerAUID, pinTestCorrectPin)
	requireCode(t, apiErr, apierror.ErrorCodeInsufficientPerms)
	assert.Empty(t, f.pins.pins)
}

func TestSetTemporarySigningPin_AccountUserOutsideAccount(t *testing.T) {
	t.Parallel()
	f := newSigningPinFixture(t)
	f.users.EXPECT().FindAccountUser(gomock.Any(), pinTestAccountID, pinTestSignerAUID).
		Return(nil, apierror.NewResourceNotFoundError("Account user not found."))

	apiErr := f.svc.SetTemporarySigningPin(pinCtx(pinIdentity(constants.RoleTypeCustom, teamUpdate)), pinTestSignerAUID, pinTestCorrectPin)
	requireCode(t, apiErr, apierror.ErrorCodeResourceNotFound)
	assert.Empty(t, f.pins.pins)
}

func TestClearSigningPin(t *testing.T) {
	t.Parallel()
	f := newSigningPinFixture(t)
	f.expectSigner(pinTestAccountID, "active")
	f.storePin(t, pinTestCorrectPin, false)
	admin := pinCtx(pinIdentity(constants.RoleTypeCustom, teamUpdate))

	requireCode(t, f.svc.ClearSigningPin(pinCtx(pinIdentity(constants.RoleTypeCustom)), pinTestSignerAUID), apierror.ErrorCodeInsufficientPerms)

	require.Nil(t, f.svc.ClearSigningPin(admin, pinTestSignerAUID))
	assert.Empty(t, f.pins.pins)
	require.Nil(t, f.svc.ClearSigningPin(admin, pinTestSignerAUID))
	assert.Equal(t, []string{"cleared"}, f.outbox.signingPinEvents())

	_, apiErr := f.svc.VerifySigningPin(admin, pinTestSignerAUID, pinTestCorrectPin)
	requireCode(t, apiErr, apierror.ErrorCodeSigningPINNotSet)
}

func TestSetSigningPin(t *testing.T) {
	t.Parallel()

	t.Run("replaces a locked temporary PIN with a permanent one", func(t *testing.T) {
		t.Parallel()
		f := newSigningPinFixture(t)
		hash, err := crypto.HashBcrypt(pinTestWrongPin)
		require.NoError(t, err)
		lockedUntil := f.clock.Add(time.Hour)
		f.pins.pins[pinTestCallerUserID] = domain.SigningPin{UserID: pinTestCallerUserID, PinHash: hash, IsTemporary: true, FailedAttempts: 3, LockedUntil: &lockedUntil}
		f.users.EXPECT().FindAccountUserByUserID(gomock.Any(), pinTestAccountID, pinTestCallerUserID).
			Return(&domain.SigningAccountUser{ID: pinTestCallerAUID, UserID: pinTestCallerUserID, AccountID: pinTestAccountID, StatusCode: "active"}, nil)

		require.Nil(t, f.svc.SetSigningPin(pinCtx(pinIdentity(constants.RoleTypeCustom)), pinTestCorrectPin))

		stored := f.pins.pins[pinTestCallerUserID]
		assert.False(t, stored.IsTemporary)
		assert.Nil(t, stored.LockedUntil)
		assert.Equal(t, 0, stored.FailedAttempts)
		match, err := crypto.CompareBcryptHash(stored.PinHash, pinTestCorrectPin)
		require.NoError(t, err)
		assert.True(t, match)
		assert.Equal(t, []string{"set"}, f.outbox.signingPinEvents())
		assert.Equal(t, pinTestCallerAUID, f.outbox.events[0]["resource_id"])
		assert.Equal(t, string(constants.ObjectTypeSigningPIN), f.outbox.events[0]["resource_type"])
	})

	t.Run("refuses a scanning station login", func(t *testing.T) {
		t.Parallel()
		f := newSigningPinFixture(t)
		requireCode(t, f.svc.SetSigningPin(pinCtx(pinIdentity(constants.RoleTypeScanner)), pinTestCorrectPin), apierror.ErrorCodeInsufficientPerms)
	})

	t.Run("refuses an API key", func(t *testing.T) {
		t.Parallel()
		f := newSigningPinFixture(t)
		identity := pinIdentity(constants.RoleTypeAdmin)
		identity.Type = types.IdentityActorTypeAPIKey
		requireCode(t, f.svc.SetSigningPin(pinCtx(identity), pinTestCorrectPin), apierror.ErrorCodeInsufficientPerms)
	})

	t.Run("refuses an agent", func(t *testing.T) {
		t.Parallel()
		f := newSigningPinFixture(t)
		identity := pinIdentity(constants.RoleTypeAdmin)
		identity.Type = types.IdentityActorTypeAgent
		requireCode(t, f.svc.SetSigningPin(pinCtx(identity), pinTestCorrectPin), apierror.ErrorCodeInsufficientPerms)
	})

	t.Run("refuses a trivial PIN", func(t *testing.T) {
		t.Parallel()
		f := newSigningPinFixture(t)
		apiErr := f.svc.SetSigningPin(pinCtx(pinIdentity(constants.RoleTypeCustom)), "9876")
		requireCode(t, apiErr, apierror.ErrorCodeValidationFailed)
		assert.Equal(t, "pin", apiErr.Param)
	})
}

func TestSigningPinAuditEvents_NeverCarryThePin(t *testing.T) {
	t.Parallel()
	f := newSigningPinFixture(t)
	f.expectSigner(pinTestAccountID, "active")
	admin := pinCtx(pinIdentity(constants.RoleTypeCustom, teamUpdate))

	require.Nil(t, f.svc.SetTemporarySigningPin(admin, pinTestSignerAUID, pinTestCorrectPin))
	for range 5 {
		_, _ = f.svc.VerifySigningPin(admin, pinTestSignerAUID, pinTestWrongPin)
	}
	f.clock = f.clock.Add(time.Hour)
	require.Nil(t, f.svc.ChangeSigningPin(admin, pinTestSignerAUID, pinTestCorrectPin, pinTestReplacement))
	require.Nil(t, f.svc.ClearSigningPin(admin, pinTestSignerAUID))

	raw, err := json.Marshal(f.outbox.events)
	require.NoError(t, err)
	for _, pin := range []string{pinTestCorrectPin, pinTestWrongPin, pinTestReplacement} {
		assert.NotContains(t, string(raw), pin)
	}
	assert.NotContains(t, string(raw), "$2a$")
	assert.Len(t, f.outbox.events, 9)
}
