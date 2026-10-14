package resourceregistry

import (
	"context"
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeDemandOverride,
		Load:       resourceloaders.LoadDemandOverrides,
		Subs: []include.SubField{
			// The scope is polymorphic (item or product line), so it resolves to an Entity stashed upstream rather than through a typed loader — a SubField can only name one Target, and either typed include would be null for half the rows.
			{Key: "scope", Cardinality: include.CardinalityOnePtr, Populate: populateScopeOnDemandOverride},
			{
				Key:         "unit",
				Target:      constants.ObjectTypeUnit,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractDemandOverrideRefIDs("unit_id"),
				Populate:    populateDemandOverrideUnit,
			},
			{
				Key:         "created_by",
				Target:      constants.ObjectTypeActor,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractDemandOverrideRefIDs("created_by_id"),
				Populate:    populateDemandOverrideCreatedBy,
			},
		},
	})
}

func populateScopeOnDemandOverride(ctx context.Context, parent any, _ map[string]any) {
	o, ok := parent.(*apiresource.DemandOverride)
	if !ok {
		return
	}
	v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeDemandOverride, o.ID, "scope")
	if !ok {
		return
	}
	o.Scope = v.(*apiresource.Entity)
}

// demandOverrideRefID reads one stashed FK id off LoadMeta. The unit and created_by expandables are plain single-id references, so extraction and population differ only by which key they read.
func demandOverrideRefID(ctx context.Context, parent any, key string) string {
	o, ok := parent.(*apiresource.DemandOverride)
	if !ok {
		return ""
	}
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeDemandOverride, o.ID, key)
	return id
}

func extractDemandOverrideRefIDs(key string) func(context.Context, any) []string {
	return func(ctx context.Context, parent any) []string {
		id := demandOverrideRefID(ctx, parent, key)
		if id == "" {
			return nil
		}
		return []string{id}
	}
}

func populateDemandOverrideUnit(ctx context.Context, parent any, loaded map[string]any) {
	o, ok := parent.(*apiresource.DemandOverride)
	if !ok {
		return
	}
	if v, ok := loaded[demandOverrideRefID(ctx, parent, "unit_id")]; ok {
		o.Unit = v.(*apiresource.Unit)
	}
}

func populateDemandOverrideCreatedBy(ctx context.Context, parent any, loaded map[string]any) {
	o, ok := parent.(*apiresource.DemandOverride)
	if !ok {
		return
	}
	if v, ok := loaded[demandOverrideRefID(ctx, parent, "created_by_id")]; ok {
		o.CreatedBy = v.(*apiresource.Actor)
	}
}
