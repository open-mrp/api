package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/repository"
	apierror "github.com/open-mrp/api/shared/errors"

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
	inboxRepo.EXPECT().GetByAddress(gomock.Any(), "support@openmrp-test.com").
		Return(nil, apierror.NewResourceNotFoundError("nope"))

	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewEmailInboxRepo().Return(inboxRepo).AnyTimes()
	svc := &conversationSvcImpl{repoFactory: factory}

	// Unknown inbox is dropped (acked) without error so the SQS message isn't retried forever.
	apiErr := svc.IngestInboundEmail(context.Background(), domain.IngestInboundEmailInput{
		Recipients:   []string{"support@openmrp-test.com"},
		RfcMessageID: "m1@x",
	})
	assert.Nil(t, apiErr)
}

func TestIngestInboundEmail_ResolvesForwardingAddress(t *testing.T) {
	ctrl := gomock.NewController(t)
	inboxRepo := repositorymock.NewMockEmailInboxRepo(ctrl)
	// A forwarded email: the forward target (<inbox_id>@inbound.openmrp.ai) lands in Delivered-To and
	// resolves the inbox by id — even though the customer's real inbox address (testing@acme.com) is
	// all that survives in the To header. GetByAddress is never consulted for the matching candidate.
	inboxRepo.EXPECT().GetByIDSystem(gomock.Any(), "emix_1").
		Return(&domain.EmailInbox{ID: "emix_1", AccountID: "ac_1", Address: "testing@acme.com", Status: domain.EmailInboxStatusActive}, nil)

	emailMsgRepo := repositorymock.NewMockEmailMessageRepo(ctrl)
	// Short-circuit on the dedup guard so the test targets resolution, not the full threading path.
	emailMsgRepo.EXPECT().GetByRfcID(gomock.Any(), "fwd@x").
		Return(&domain.EmailMessage{ID: "emmg_1"}, nil)

	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewEmailInboxRepo().Return(inboxRepo).AnyTimes()
	factory.EXPECT().NewEmailMessageRepo().Return(emailMsgRepo).AnyTimes()
	svc := &conversationSvcImpl{repoFactory: factory, inboundEmailDomain: "inbound.openmrp.ai"}

	apiErr := svc.IngestInboundEmail(context.Background(), domain.IngestInboundEmailInput{
		Recipients:   []string{"emix_1@inbound.openmrp.ai", "testing@acme.com"},
		RfcMessageID: "fwd@x",
	})
	require.Nil(t, apiErr)
}

func TestCreateEmailThreadConversation_SeatsGroupMembersDeduped(t *testing.T) {
	ctrl := gomock.NewController(t)

	convRepo := repositorymock.NewMockConversationRepo(ctrl)
	convRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), "ac_1").Return(nil)
	convRepo.EXPECT().BindInbox(gomock.Any(), gomock.Any(), "ac_1", "emix_1", "cust@out.com").Return(nil)
	convRepo.EXPECT().SetWorkflowStatus(gomock.Any(), gomock.Any(), "ac_1", gomock.Any()).Return(nil)

	partRepo := repositorymock.NewMockParticipantRepo(ctrl)
	var seatedUsers []string
	var seatedAgents []string
	partRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, p *domain.ConversationParticipant) *apierror.APIError {
			require.NotNil(t, p.AccountUserID)
			seatedUsers = append(seatedUsers, *p.AccountUserID)
			return nil
		}).AnyTimes()
	partRepo.EXPECT().CreateAgent(gomock.Any(), gomock.Any(), "ac_1", gomock.Any()).DoAndReturn(
		func(_ context.Context, _, _ string, in *domain.AddAgentParticipantInput) *apierror.APIError {
			seatedAgents = append(seatedAgents, in.AgentConfigID)
			return nil
		}).AnyTimes()

	groupRepo := repositorymock.NewMockMessagingGroupRepo(ctrl)
	// The roster carries two humans and two agents — one of which duplicates the inbox's own triage agent.
	groupRepo.EXPECT().ListMembers(gomock.Any(), "mggp_1").Return([]*domain.MessagingGroupMember{
		{MemberType: domain.MessagingGroupMemberTypeUser, AccountUserID: new("acus_1")},
		{MemberType: domain.MessagingGroupMemberTypeUser, AccountUserID: new("acus_2")},
		{MemberType: domain.MessagingGroupMemberTypeAgent, AgentConfigID: new("agdf_inbox")},
		{MemberType: domain.MessagingGroupMemberTypeAgent, AgentConfigID: new("agdf_extra")},
	}, nil)

	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewConversationRepo().Return(convRepo).AnyTimes()
	factory.EXPECT().NewParticipantRepo().Return(partRepo).AnyTimes()
	factory.EXPECT().NewMessagingGroupRepo().Return(groupRepo).AnyTimes()

	svc := &conversationSvcImpl{}
	inbox := &domain.EmailInbox{
		ID:            "emix_1",
		AccountID:     "ac_1",
		Address:       "support@x.com",
		Status:        domain.EmailInboxStatusActive,
		AgentConfigID: new("agdf_inbox"),
		GroupID:       new("mggp_1"),
	}
	convID, apiErr := svc.createEmailThreadConversation(context.Background(), factory, inbox, domain.IngestInboundEmailInput{
		From:    "cust@out.com",
		Subject: "Help",
	})
	require.Nil(t, apiErr)
	require.NotEmpty(t, convID)

	// Both humans seated; the inbox's triage agent seated exactly once (the roster's duplicate collapses),
	// plus the roster's extra agent — three participants for four member rows + the inbox agent.
	assert.ElementsMatch(t, []string{"acus_1", "acus_2"}, seatedUsers)
	assert.ElementsMatch(t, []string{"agdf_inbox", "agdf_extra"}, seatedAgents)
}

func TestExternalSenderMetadata_RoundTrip(t *testing.T) {
	meta := marshalExternalSenderMeta("  Jane Doe  ", "  jane@x.com  ")
	name, addr := externalSenderFromMetadata(meta)
	assert.Equal(t, "Jane Doe", name)
	assert.Equal(t, "jane@x.com", addr)
	assert.Equal(t, "Jane Doe <jane@x.com>", externalSenderLabel(name, addr))

	assert.Equal(t, "bob@x.com", externalSenderLabel("", "bob@x.com"), "address-only falls back to the address")
	assert.Nil(t, marshalExternalSenderMeta("", ""), "no sender → no metadata (column stays NULL)")

	n, a := externalSenderFromMetadata(nil)
	assert.Empty(t, n)
	assert.Empty(t, a)
}

func TestResolveSenders_ExternalEmailAttribution(t *testing.T) {
	ctrl := gomock.NewController(t)
	// An inbound email has no participant author; the sender lives on the message metadata, not a roster row.
	participantRepo := repositorymock.NewMockParticipantRepo(ctrl)
	participantRepo.EXPECT().ListAll(gomock.Any(), "cv_x").
		Return([]*domain.ConversationParticipant{}, nil).AnyTimes()
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewParticipantRepo().Return(participantRepo).AnyTimes()
	svc := &conversationSvcImpl{repoFactory: factory}

	msg := &domain.Message{
		ID:             "mg_x",
		ConversationID: "cv_x",
		Metadata:       marshalExternalSenderMeta("Carl Customer", "carl@customer.com"),
	}
	svc.resolveSenders(context.Background(), "cv_x", "ac_x", []*domain.Message{msg}, false)
	require.NotNil(t, msg.SenderDisplayName, "the external email sender must be surfaced as a display name")
	assert.Equal(t, "Carl Customer <carl@customer.com>", *msg.SenderDisplayName)
}

func TestIngestInboundEmail_DedupAlreadyIngested(t *testing.T) {
	ctrl := gomock.NewController(t)
	inboxRepo := repositorymock.NewMockEmailInboxRepo(ctrl)
	inboxRepo.EXPECT().GetByAddress(gomock.Any(), "support@openmrp-test.com").
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
		Recipients:   []string{"support@openmrp-test.com"},
		RfcMessageID: "dup@x",
	})
	require.Nil(t, apiErr)
}
