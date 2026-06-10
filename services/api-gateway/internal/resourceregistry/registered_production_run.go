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

// productionRunID returns the run ID for either resource shape that shares the
// production_run object type (detail on Get/Create/Update, summary on List).
func productionRunID(parent any) string {
	switch pr := parent.(type) {
	case *apiresource.ProductionRunDetail:
		return pr.ID
	case *apiresource.ProductionRunSummary:
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
	switch pr := parent.(type) {
	case *apiresource.ProductionRunDetail:
		pr.ResponsibleUser = v.(*apiresource.AccountUser)
	case *apiresource.ProductionRunSummary:
		pr.ResponsibleUser = v.(*apiresource.AccountUser)
	}
}
