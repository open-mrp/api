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
			{
				Key:         "responsible_user",
				Target:      constants.ObjectTypeAccountUser,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractResponsibleUserIDFromProductionRun,
				Populate:    populateResponsibleUserOnProductionRun,
			},
		},
	})
}

// productionRunID returns the run ID for the production_run resource.
func productionRunID(parent any) string {
	if pr, ok := parent.(*apiresource.ProductionRun); ok {
		return pr.ID
	}
	return ""
}

func extractResponsibleUserIDFromProductionRun(ctx context.Context, parent any) []string {
	runID := productionRunID(parent)
	if runID == "" {
		return nil
	}
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProductionRun, runID, "responsible_user_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateResponsibleUserOnProductionRun(ctx context.Context, parent any, loaded map[string]any) {
	runID := productionRunID(parent)
	if runID == "" {
		return
	}
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProductionRun, runID, "responsible_user_id")
	if id == "" {
		return
	}
	v, ok := loaded[id]
	if !ok {
		return
	}
	if pr, ok := parent.(*apiresource.ProductionRun); ok {
		pr.ResponsibleUser = v.(*apiresource.AccountUser)
	}
}
