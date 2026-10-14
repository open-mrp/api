package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/include"
)

func init() {
	// AgentMemory is a leaf with an inline (non-expandable) Entity reference. Empty-Subs Definition; the loader builds Entity inline from the proto's entity_type/entity_id pair. First resource registered on AgentService.
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeAgentMemory,
		Load:       resourceloaders.LoadAgentMemories,
	})
}
