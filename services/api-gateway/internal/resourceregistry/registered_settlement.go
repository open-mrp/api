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
		ObjectType: constants.ObjectTypeSettlement,
		Load:       resourceloaders.LoadSettlements,
		Subs: []resourcekit.SubField{
			{
				Key:         "responsible_user",
				Target:      constants.ObjectTypeAccountUser,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractResponsibleUserIDFromSettlement,
				Populate:    populateResponsibleUserOnSettlement,
			},
			{Key: "allocations", Cardinality: resourcekit.CardinalityList, Target: constants.ObjectTypeTransactionAllocation, ExtractRefs: extractAllocationRefsFromSettlement, Populate: populateAllocationsOnSettlement},
		},
	})
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeSettlementSummary,
		Load:       resourceloaders.LoadSettlementSummaries,
	})
}

func extractResponsibleUserIDFromSettlement(ctx context.Context, parent any) []string {
	s := parent.(*apiresource.Settlement)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeSettlement, s.ID, "responsible_user_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateResponsibleUserOnSettlement(ctx context.Context, parent any, loaded map[string]any) {
	s := parent.(*apiresource.Settlement)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeSettlement, s.ID, "responsible_user_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		s.ResponsibleUser = v.(*apiresource.AccountUser)
	}
}

func populateAllocationsOnSettlement(ctx context.Context, parent any, _ map[string]any) {
	s := parent.(*apiresource.Settlement)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeSettlement, s.ID, "allocations")
	if !ok {
		return
	}
	s.Allocations = v.(*apiresource.List[apiresource.TransactionAllocation])
}

// The resolver runs Populate before gathering refs, so the allocations are already on the settlement.
func extractAllocationRefsFromSettlement(_ context.Context, parent any) []any {
	s := parent.(*apiresource.Settlement)
	if s.Allocations == nil {
		return nil
	}
	refs := make([]any, len(s.Allocations.Data))
	for i := range s.Allocations.Data {
		refs[i] = &s.Allocations.Data[i]
	}
	return refs
}
