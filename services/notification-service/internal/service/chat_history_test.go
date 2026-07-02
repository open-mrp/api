package service

import (
	"testing"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/shared/messaging"
	"github.com/stretchr/testify/assert"
)

// chatHistoryForAgent must stamp the dispatched agent's own past replies as "assistant", a person's
// turn as a named "user", and a *different* agent's turn as a "user" carrying its config id (Name
// resolved downstream in agent-service) — never as the dispatched agent's own assistant turn.
func TestChatHistoryForAgent_RolesAndAttribution(t *testing.T) {
	const me = "agdf_me"
	entries := []chatHistoryEntry{
		{name: "Alice", body: "hey groot"},                 // person
		{agentConfigID: me, body: "hi, I'm groot"},         // dispatched agent's own reply
		{agentConfigID: "agdf_other", body: "groot rocks"}, // a different agent
		{body: "anon person"},                              // person with unresolved name
	}

	got := chatHistoryForAgent(entries, me)

	want := []messaging.ChatHistoryMessage{
		{Role: "user", Name: "Alice", Body: "hey groot"},
		{Role: "assistant", Body: "hi, I'm groot"},
		{Role: "user", AgentConfigID: "agdf_other", Body: "groot rocks"},
		{Role: "user", Name: "", Body: "anon person"},
	}
	assert.Equal(t, want, got)
}

func TestChatHistoryForAgent_EmptyReturnsNil(t *testing.T) {
	assert.Nil(t, chatHistoryForAgent(nil, "agdf_me"))
}

// describeBodylessMessage renders a short stand-in for link/attachment-only messages so they aren't
// invisible in the agent's thread context; a link's resource type wins over the generic preview.
func TestDescribeBodylessMessage(t *testing.T) {
	tests := []struct {
		name string
		msg  *domain.Message
		want string
	}{
		{
			name: "resource link prefers humanized type over preview",
			msg:  &domain.Message{LinkResourceType: strPtr("sales_order"), Preview: strPtr("🔗 Link")},
			want: "[shared a sales order link]",
		},
		{
			name: "attachment-only falls back to persisted preview",
			msg:  &domain.Message{Preview: strPtr("📎 Attachment")},
			want: "[📎 Attachment]",
		},
		{
			name: "nothing to describe",
			msg:  &domain.Message{},
			want: "",
		},
		{
			name: "blank preview is not described",
			msg:  &domain.Message{Preview: strPtr("   ")},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, describeBodylessMessage(tt.msg))
		})
	}
}
