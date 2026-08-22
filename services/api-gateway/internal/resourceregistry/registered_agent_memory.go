package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	// AgentMemory is a leaf with an inline (non-expandable) Entity reference. Empty-Subs Definition; the loader builds Entity inline from the proto's entity_type/entity_id pair. First resource registered on AgentService.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeAgentMemory,
		Load:       resourceloaders.LoadAgentMemories,
	})
}
