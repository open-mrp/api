package resourceregistry

import (
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	// AgentMemory is a leaf with an inline (non-expandable) Entity reference. Empty-Subs Definition; the loader builds Entity inline from the proto's entity_type/entity_id pair. First resource registered on AgentService.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeAgentMemory,
		Load:       resourceloaders.LoadAgentMemories,
	})
}
