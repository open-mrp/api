package resourceregistry

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeProductionRun,
		Load:       resourceloaders.LoadProductionRuns,
		Subs: []resourcekit.SubField{
			{Key: "responsible_user", Populate: populateResponsibleUserOnProductionRun},
		},
	})
}

func populateResponsibleUserOnProductionRun(ctx context.Context, parent any, _ map[string]any) {
	pr := parent.(*apiresource.ProductionRunDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeProductionRun, pr.ID, "responsible_user")
	if !ok {
		return
	}
	pr.ResponsibleUser = v.(*apiresource.AccountUser)
}
