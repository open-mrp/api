package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/factory"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestPublishChatMessageCreated_RootsAtConversationWithoutContent(t *testing.T) {
	ctrl := gomock.NewController(t)
	outbox := &ingestOutboxRepo{}
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewOutboxRepo().Return(outbox)

	body := "quote for the order"
	subject := "Re: order"
	msg := &domain.Message{
		ID:                  "mg_1",
		ConversationID:      "cv_1",
		AccountID:           "ac_1",
		Status:              string(constants.MessageStatusSent),
		Visibility:          string(constants.MessageVisibilityInternal),
		SenderParticipantID: new("cvpt_1"),
		Channel:             new(string(constants.MessageChannelMessage)),
		Body:                &body,
		Subject:             &subject,
		Preview:             &body,
		LinkResourceType:    new(string(constants.ObjectTypeSalesOrder)),
		LinkResourceID:      new("so_1"),
	}
	ctx := appctx.WithSystemIdentity(context.Background(), constants.SystemActorEmailIngest, "ac_1")
	require.Nil(t, publishChatMessageCreated(ctx, factory, msg))
	require.Len(t, outbox.created, 1)

	var event audit.PublishedEvent
	require.NoError(t, json.Unmarshal(outbox.created[0].Payload.Data, &event))
	assert.Equal(t, constants.AuditActionCreate, event.Action)
	assert.Equal(t, constants.ObjectTypeChatMessage, event.ResourceType)
	assert.Equal(t, "mg_1", event.ResourceID)
	assert.Equal(t, constants.ObjectTypeConversation, event.RootResourceType)
	assert.Equal(t, "cv_1", event.RootResourceID)

	fields := map[string]string{}
	for _, c := range event.Changes {
		fields[c.Field] = string(c.NewValue)
	}
	assert.Equal(t, `"cv_1"`, fields["conversation_id"])
	assert.Equal(t, `"sent"`, fields["status"])
	assert.Equal(t, `"internal"`, fields["visibility"])
	raw := string(outbox.created[0].Payload.Data)
	assert.NotContains(t, raw, body)
	assert.NotContains(t, raw, subject)
	assert.NotContains(t, raw, "so_1")
}
