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
		ObjectType: constants.ObjectTypeTerritory,
		Load:       resourceloaders.LoadTerritories,
		Subs: []resourcekit.SubField{
			{
				Key:         "sales_rep",
				Target:      constants.ObjectTypeAccountUser,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractSalesRepIDFromTerritory,
				Populate:    populateSalesRepOnTerritory,
			},
			{
				Key:         "product_line",
				Target:      constants.ObjectTypeProductLine,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractProductLineIDFromTerritory,
				Populate:    populateProductLineOnTerritory,
			},
		},
	})
}

func extractSalesRepIDFromTerritory(ctx context.Context, parent any) []string {
	t := parent.(*apiresource.Territory)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeTerritory, t.ID, "sales_rep_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateSalesRepOnTerritory(ctx context.Context, parent any, loaded map[string]any) {
	t := parent.(*apiresource.Territory)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeTerritory, t.ID, "sales_rep_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		t.SalesRep = v.(*apiresource.AccountUser)
	}
}

func extractProductLineIDFromTerritory(ctx context.Context, parent any) []string {
	t := parent.(*apiresource.Territory)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeTerritory, t.ID, "product_line_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateProductLineOnTerritory(ctx context.Context, parent any, loaded map[string]any) {
	t := parent.(*apiresource.Territory)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeTerritory, t.ID, "product_line_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		t.ProductLine = v.(*apiresource.ProductLine)
	}
}
