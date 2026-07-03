package service

import (
	"context"
	"testing"

	"github.com/augno/api/services/notification-service/internal/domain"
	factorymock "github.com/augno/api/services/notification-service/internal/domain/mock/factory"
	repositorymock "github.com/augno/api/services/notification-service/internal/domain/mock/repository"
	apierror "github.com/augno/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestThreadCandidates_OrderAndDedup(t *testing.T) {
	// In-Reply-To leads; References follow; duplicates collapse; blanks dropped.
	got := threadCandidates("b@x", []string{"a@x", "b@x", "", "c@x"})
	assert.Equal(t, []string{"b@x", "a@x", "c@x"}, got)

	assert.Nil(t, threadCandidates("", nil))
}

func TestIngestInboundEmail_UnknownInboxDropped(t *testing.T) {
	ctrl := gomock.NewController(t)
	inboxRepo := repositorymock.NewMockEmailInboxRepo(ctrl)
	inboxRepo.EXPECT().GetByAddress(gomock.Any(), "support@augno-test.com").
		Return(nil, apierror.NewResourceNotFoundError("nope"))

	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewEmailInboxRepo().Return(inboxRepo).AnyTimes()
	svc := &conversationSvcImpl{repoFactory: factory}

	// Unknown inbox is dropped (acked) without error so the SQS message isn't retried forever.
	apiErr := svc.IngestInboundEmail(context.Background(), domain.IngestInboundEmailInput{
		Recipients:   []string{"support@augno-test.com"},
		RfcMessageID: "m1@x",
	})
	assert.Nil(t, apiErr)
}

func TestIngestInboundEmail_ResolvesForwardingAddress(t *testing.T) {
	ctrl := gomock.NewController(t)
	inboxRepo := repositorymock.NewMockEmailInboxRepo(ctrl)
	// A forwarded email: the forward target (<inbox_id>@inbound.augno.com) lands in Delivered-To and
	// resolves the inbox by id — even though the customer's real inbox address (testing@sellerco.com) is
	// all that survives in the To header. GetByAddress is never consulted for the matching candidate.
	inboxRepo.EXPECT().GetByIDSystem(gomock.Any(), "emix_1").
		Return(&domain.EmailInbox{ID: "emix_1", AccountID: "ac_1", Address: "testing@sellerco.com", Status: domain.EmailInboxStatusActive}, nil)

	emailMsgRepo := repositorymock.NewMockEmailMessageRepo(ctrl)
	// Short-circuit on the dedup guard so the test targets resolution, not the full threading path.
	emailMsgRepo.EXPECT().GetByRfcID(gomock.Any(), "fwd@x").
		Return(&domain.EmailMessage{ID: "emmg_1"}, nil)

	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewEmailInboxRepo().Return(inboxRepo).AnyTimes()
	factory.EXPECT().NewEmailMessageRepo().Return(emailMsgRepo).AnyTimes()
	svc := &conversationSvcImpl{repoFactory: factory, inboundEmailDomain: "inbound.augno.com"}

	apiErr := svc.IngestInboundEmail(context.Background(), domain.IngestInboundEmailInput{
		Recipients:   []string{"emix_1@inbound.augno.com", "testing@sellerco.com"},
		RfcMessageID: "fwd@x",
	})
	require.Nil(t, apiErr)
}

func TestIngestInboundEmail_DedupAlreadyIngested(t *testing.T) {
	ctrl := gomock.NewController(t)
	inboxRepo := repositorymock.NewMockEmailInboxRepo(ctrl)
	inboxRepo.EXPECT().GetByAddress(gomock.Any(), "support@augno-test.com").
		Return(&domain.EmailInbox{ID: "emix_1", AccountID: "ac_1", Status: domain.EmailInboxStatusActive}, nil)

	emailMsgRepo := repositorymock.NewMockEmailMessageRepo(ctrl)
	emailMsgRepo.EXPECT().GetByRfcID(gomock.Any(), "dup@x").
		Return(&domain.EmailMessage{ID: "emmg_1"}, nil)

	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewEmailInboxRepo().Return(inboxRepo).AnyTimes()
	factory.EXPECT().NewEmailMessageRepo().Return(emailMsgRepo).AnyTimes()
	svc := &conversationSvcImpl{repoFactory: factory}

	// A redelivered email (rfc id already in the ledger) is acked with no further work — no tx, no
	// thread lookup, no message create.
	apiErr := svc.IngestInboundEmail(context.Background(), domain.IngestInboundEmailInput{
		Recipients:   []string{"support@augno-test.com"},
		RfcMessageID: "dup@x",
	})
	require.Nil(t, apiErr)
}
