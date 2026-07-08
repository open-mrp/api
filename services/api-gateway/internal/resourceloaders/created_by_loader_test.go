package resourceloaders

import (
	"testing"

	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/platform"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatedByFromAuditActor(t *testing.T) {
	t.Parallel()

	agentNames := map[string]AgentDefinitionName{
		"adef_1": {Name: "Ops Agent", Slug: "ops-agent"},
	}

	t.Run("nil actor resolves to system", func(t *testing.T) {
		cb := createdByFromAuditActor(nil, nil)
		assert.Equal(t, constants.CreatedByRelationSystem, cb.Relation)
		assert.Nil(t, cb.Actor)
	})

	t.Run("internal user keeps audit-resolved name", func(t *testing.T) {
		cb := createdByFromAuditActor(&pb.AuditActor{
			Id:        "us_1",
			Type:      string(constants.CreatedByRelationInternal),
			ActorType: string(constants.ActorTypeUser),
			Name:      new("Jane Doe"),
		}, agentNames)
		require.NotNil(t, cb.Actor)
		assert.Equal(t, constants.CreatedByRelationInternal, cb.Relation)
		assert.Equal(t, constants.ActorTypeUser, cb.Actor.Type)
		require.NotNil(t, cb.Actor.Name)
		assert.Equal(t, "Jane Doe", *cb.Actor.Name)
	})

	t.Run("internal agent gets name from agent-service resolution", func(t *testing.T) {
		// The audit store cannot resolve agent names (agent definitions live in the
		// agent-service), so the actor arrives nameless and must be filled from the
		// resolved map — this is the "order created by an agent shows Unknown" bug.
		cb := createdByFromAuditActor(&pb.AuditActor{
			Id:        "adef_1",
			Type:      string(constants.CreatedByRelationInternal),
			ActorType: string(constants.ActorTypeAgent),
		}, agentNames)
		require.NotNil(t, cb.Actor)
		assert.Equal(t, constants.CreatedByRelationInternal, cb.Relation)
		assert.Equal(t, constants.ActorTypeAgent, cb.Actor.Type)
		require.NotNil(t, cb.Actor.Name)
		assert.Equal(t, "Ops Agent", *cb.Actor.Name)
	})

	t.Run("unresolvable agent keeps nil name rather than failing", func(t *testing.T) {
		cb := createdByFromAuditActor(&pb.AuditActor{
			Id:        "adef_missing",
			Type:      string(constants.CreatedByRelationInternal),
			ActorType: string(constants.ActorTypeAgent),
		}, agentNames)
		require.NotNil(t, cb.Actor)
		assert.Nil(t, cb.Actor.Name)
	})

	t.Run("audit-resolved name is not overridden by the agent map", func(t *testing.T) {
		cb := createdByFromAuditActor(&pb.AuditActor{
			Id:        "adef_1",
			Type:      string(constants.CreatedByRelationInternal),
			ActorType: string(constants.ActorTypeAgent),
			Name:      new("Pre-resolved"),
		}, agentNames)
		require.NotNil(t, cb.Actor)
		require.NotNil(t, cb.Actor.Name)
		assert.Equal(t, "Pre-resolved", *cb.Actor.Name)
	})

	t.Run("non internal or customer relation resolves to system", func(t *testing.T) {
		cb := createdByFromAuditActor(&pb.AuditActor{
			Id:        "ak_1",
			Type:      "supplier",
			ActorType: string(constants.ActorTypeAPIKey),
		}, nil)
		assert.Equal(t, constants.CreatedByRelationSystem, cb.Relation)
		assert.Nil(t, cb.Actor)
	})

	t.Run("customer relation maps with actor", func(t *testing.T) {
		cb := createdByFromAuditActor(&pb.AuditActor{
			Id:        "us_2",
			Type:      string(constants.CreatedByRelationCustomer),
			ActorType: string(constants.ActorTypeUser),
			Name:      new("Buyer"),
		}, nil)
		require.NotNil(t, cb.Actor)
		assert.Equal(t, constants.CreatedByRelationCustomer, cb.Relation)
	})
}

func TestLoadCreatorAgentNames_CollectsOnlyNamelessAgents(t *testing.T) {
	t.Parallel()

	// No agent creators → no agent-service round trip, nil map.
	names := loadCreatorAgentNames(t.Context(), []*pb.ResourceCreator{
		{ResourceId: "or_1", Actor: &pb.AuditActor{Id: "us_1", ActorType: string(constants.ActorTypeUser), Type: "internal"}},
		{ResourceId: "or_2", Actor: &pb.AuditActor{Id: "adef_named", ActorType: string(constants.ActorTypeAgent), Type: "internal", Name: new("Already Named")}},
		{ResourceId: "or_3", Actor: nil},
	})
	assert.Nil(t, names)

	// Sanity-check the resulting CreatedBy for the named agent is preserved end to end.
	cb := createdByFromAuditActor(&pb.AuditActor{
		Id:        "adef_named",
		ActorType: string(constants.ActorTypeAgent),
		Type:      "internal",
		Name:      new("Already Named"),
	}, names)
	require.NotNil(t, cb.Actor)
	require.NotNil(t, cb.Actor.Name)
	assert.Equal(t, "Already Named", *cb.Actor.Name)
}
