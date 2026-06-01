package resourceregistry

import (
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	// EDIRun is the purest leaf — no nested fields at all.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeEDIRun,
		Load:       resourceloaders.LoadEDIRuns,
	})
}
