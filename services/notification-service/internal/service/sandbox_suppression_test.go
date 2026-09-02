package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/notification-service/internal/domain"
	repositorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/repository"
	servicemock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/service"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Sandbox suppression, which gates whether a tenant's mail reaches its real counterparties.
//
// The guard used to read AccountMode off the identity on the context. That worked for a send made
// inside an HTTP request and failed silently for every other path: a message published from the
// dashboard's outbox carries an account id and no AccountMode, and one republished by a consumer
// carried no identity at all. Sandbox invoices, acknowledgements, purchase orders and statements
// went out to live customers. These pin the behavior to the account instead.

type sandboxHarness struct {
	svc         domain.NotificationSvc
	accountRepo *repositorymock.MockAccountRepo
	emailLog    *repositorymock.MockEmailLogRepo
	sender      *servicemock.MockEmailSender
}

func newSandboxHarness(t *testing.T) *sandboxHarness {
	t.Helper()
	ctrl := gomock.NewController(t)

	h := &sandboxHarness{
		accountRepo: repositorymock.NewMockAccountRepo(ctrl),
		emailLog:    repositorymock.NewMockEmailLogRepo(ctrl),
		sender:      servicemock.NewMockEmailSender(ctrl),
	}
	h.svc = NewNotificationSvc(&NotificationSvcConfig{
		EmailLogRepo:     h.emailLog,
		AccountRepo:      h.accountRepo,
		EmailSender:      h.sender,
		TemplateRenderer: &stubTemplateRenderer{},
	})
	return h
}

func sendData(accountID *string) domain.EmailSendData {
	return domain.EmailSendData{
		To:        []string{"customer@example.com"},
		Subject:   "Invoice 000123",
		Body:      "<html>invoice</html>",
		AccountID: accountID,
	}
}

func accountID(s string) *string { return &s }

// The case the security review found: a document email published by a consumer arrives with no
// identity on the context at all. Before, that meant no suppression and a live send.
func TestSendEmail_SuppressesSandboxWithNoIdentityOnContext(t *testing.T) {
	t.Parallel()
	h := newSandboxHarness(t)

	h.accountRepo.EXPECT().IsSandbox(gomock.Any(), "ac_sandbox").Return(true, nil)
	h.emailLog.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	// The assertion that matters: SES is never called.
	h.sender.EXPECT().Send(gomock.Any(), gomock.Any()).Times(0)

	id, apiErr := h.svc.SendEmail(context.Background(), sendData(accountID("ac_sandbox")))
	require.Nil(t, apiErr)
	require.NotNil(t, id, "a suppressed send still records a log row, as the legacy sender did")
}

// The dashboard's outbox writes an identity carrying only Target.AccountID. An AccountMode-based
// check reads that as production.
func TestSendEmail_SuppressesSandboxWhenIdentityOmitsAccountMode(t *testing.T) {
	t.Parallel()
	h := newSandboxHarness(t)

	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Target: &types.IdentityTarget{AccountID: "ac_sandbox"},
	})

	h.accountRepo.EXPECT().IsSandbox(gomock.Any(), "ac_sandbox").Return(true, nil)
	h.emailLog.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	h.sender.EXPECT().Send(gomock.Any(), gomock.Any()).Times(0)

	_, apiErr := h.svc.SendEmail(ctx, sendData(accountID("ac_sandbox")))
	require.Nil(t, apiErr)
}

func TestSendEmail_SendsForALiveAccount(t *testing.T) {
	t.Parallel()
	h := newSandboxHarness(t)

	sesID := "ses-1"
	h.accountRepo.EXPECT().IsSandbox(gomock.Any(), "ac_live").Return(false, nil)
	h.sender.EXPECT().Send(gomock.Any(), gomock.Any()).Return(&sesID, nil)

	got, apiErr := h.svc.SendEmail(context.Background(), sendData(accountID("ac_live")))
	require.Nil(t, apiErr)
	require.Equal(t, &sesID, got)
}

// Platform mail — a demo request, a 5xx alert — belongs to no tenant and was never suppressed. It
// must not cost a lookup either.
func TestSendEmail_PlatformMailSkipsTheAccountLookup(t *testing.T) {
	t.Parallel()
	h := newSandboxHarness(t)

	sesID := "ses-2"
	h.accountRepo.EXPECT().IsSandbox(gomock.Any(), gomock.Any()).Times(0)
	h.sender.EXPECT().Send(gomock.Any(), gomock.Any()).Return(&sesID, nil)

	_, apiErr := h.svc.SendEmail(context.Background(), sendData(nil))
	require.Nil(t, apiErr)
}

// The guard fails open: it decides delivery, so a broken lookup must not silently swallow every
// live account's mail. The error is logged rather than returned.
func TestSendEmail_SendsWhenTheSandboxLookupFails(t *testing.T) {
	t.Parallel()
	h := newSandboxHarness(t)

	sesID := "ses-3"
	h.accountRepo.EXPECT().IsSandbox(gomock.Any(), "ac_live").
		Return(false, apierror.NewInternalError(errors.New("db down"), "lookup failed"))
	h.sender.EXPECT().Send(gomock.Any(), gomock.Any()).Return(&sesID, nil)

	_, apiErr := h.svc.SendEmail(context.Background(), sendData(accountID("ac_live")))
	require.Nil(t, apiErr)
}

// An identity that does declare sandbox mode is honored without a lookup, so an HTTP-scoped send
// keeps its fast path.
func TestSendEmail_IdentityAccountModeStillSuppresses(t *testing.T) {
	t.Parallel()
	h := newSandboxHarness(t)

	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		AccountMode: "test",
		Target:      &types.IdentityTarget{AccountID: "ac_sandbox"},
	})

	h.accountRepo.EXPECT().IsSandbox(gomock.Any(), gomock.Any()).Times(0)
	h.emailLog.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	h.sender.EXPECT().Send(gomock.Any(), gomock.Any()).Times(0)

	_, apiErr := h.svc.SendEmail(ctx, sendData(accountID("ac_sandbox")))
	require.Nil(t, apiErr)
}
