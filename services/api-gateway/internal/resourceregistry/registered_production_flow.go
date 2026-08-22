package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeProductionFlow,
		Load:       resourceloaders.LoadProductionFlows,
		Subs: []resourcekit.SubField{
			{
				Key:         "steps",
				Target:      productionFlowStepType,
				Populate:    populateStepsOnProductionFlow,
				ExtractRefs: extractStepRefsFromProductionFlow,
			},
		},
	})
}

func populateStepsOnProductionFlow(ctx context.Context, parent any, _ map[string]any) {
	flow := parent.(*apiresource.ProductionFlow)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeProductionFlow, "singleton", "steps")
	if !ok {
		return
	}
	flow.Steps = v.(*apiresource.List[apiresource.ProductionFlowStep])
}

func extractStepRefsFromProductionFlow(_ context.Context, parent any) []any {
	flow := parent.(*apiresource.ProductionFlow)
	if flow.Steps == nil {
		return nil
	}
	refs := make([]any, len(flow.Steps.Data))
	for i := range flow.Steps.Data {
		refs[i] = &flow.Steps.Data[i]
	}
	return refs
}
