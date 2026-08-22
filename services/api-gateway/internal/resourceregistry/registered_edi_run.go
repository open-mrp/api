package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	// EDIRun is the purest leaf — no nested fields at all.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeEDIRun,
		Load:       resourceloaders.LoadEDIRuns,
	})
}
