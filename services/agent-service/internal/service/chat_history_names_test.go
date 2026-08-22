package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	repositorymock "github.com/open-mrp/api/services/agent-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// resolveHistoryAgentNames resolves other-agents' display names by definition id (deduped, one lookup
// per distinct id), falls back to a generic label when a definition can't be loaded, and never looks
// up people turns or turns that already have a name.
func TestResolveHistoryAgentNames(t *testing.T) {
	ctrl := gomock.NewController(t)
	defRepo := repositorymock.NewMockAgentDefinitionRepo(ctrl)

	// "agdf_groot" resolves to its name and is looked up exactly once despite two turns (dedup).
	defRepo.EXPECT().GetByID(gomock.Any(), "agdf_groot").
		Return(&sqlc.AgentDefinition{ID: "agdf_groot", Name: "Groot"}, nil).Times(1)
	// "agdf_gone" can't be loaded → falls back to the generic label.
	defRepo.EXPECT().GetByID(gomock.Any(), "agdf_gone").
		Return(nil, apierror.NewResourceNotFoundError("not found")).Times(1)

	history := []domain.ChatHistoryMessage{
		{Role: "user", Name: "Alice", Body: "hi"},                       // person: no lookup
		{Role: "assistant", Body: "my own reply"},                       // own turn: no agent id
		{Role: "user", AgentConfigID: "agdf_groot", Body: "first"},      // other agent → "Groot"
		{Role: "user", AgentConfigID: "agdf_groot", Body: "again"},      // same agent → cached
		{Role: "user", AgentConfigID: "agdf_gone", Body: "ghost"},       // unresolvable → fallback
		{Role: "user", Name: "Bob", AgentConfigID: "agdf_x", Body: "x"}, // already named: no lookup
	}

	s := &agentDefSvcImpl{}
	s.resolveHistoryAgentNames(context.Background(), defRepo, history)

	want := []domain.ChatHistoryMessage{
		{Role: "user", Name: "Alice", Body: "hi"},
		{Role: "assistant", Body: "my own reply"},
		{Role: "user", Name: "Groot", AgentConfigID: "agdf_groot", Body: "first"},
		{Role: "user", Name: "Groot", AgentConfigID: "agdf_groot", Body: "again"},
		{Role: "user", Name: "another assistant", AgentConfigID: "agdf_gone", Body: "ghost"},
		{Role: "user", Name: "Bob", AgentConfigID: "agdf_x", Body: "x"},
	}
	assert.Equal(t, want, history)
}
