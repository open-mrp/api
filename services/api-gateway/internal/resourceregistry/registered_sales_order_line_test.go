package resourceregistry

import (
	"context"
	"testing"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

// The sales-order line's product include is driven entirely by the product_id stashed in meta — never by a prebuilt stub on the line. These tests lock that wiring: with no stashed id the product stays nil (the leak we fixed), and with a stashed id it populates from the loaded map.

func TestSOLineProductInclude_NoMeta_StaysNil(t *testing.T) {
	ctx := resourcekit.WithLoadMeta(context.Background())
	line := &apiresource.SalesOrderLine{ID: "sol_1"}

	if ids := extractProductIDFromSOLine(ctx, line); ids != nil {
		t.Errorf("extract should return nil without a stashed product_id, got %v", ids)
	}

	// Even if a product happens to be in the loaded map, an unrequested line must not pick it up.
	populateProductOnSOLine(ctx, line, map[string]any{
		"prod_1": &apiresource.Product{ID: "prod_1", Object: constants.ObjectTypeProduct},
	})
	if line.Product != nil {
		t.Errorf("line.Product must stay nil when no product_id is stashed, got %+v", line.Product)
	}
}

func TestSOLineProductInclude_WithMeta_Populates(t *testing.T) {
	ctx := resourcekit.WithLoadMeta(context.Background())
	resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeSalesOrderLine, "sol_1", "product_id", "prod_1")
	line := &apiresource.SalesOrderLine{ID: "sol_1"}

	ids := extractProductIDFromSOLine(ctx, line)
	if len(ids) != 1 || ids[0] != "prod_1" {
		t.Fatalf("extract = %v, want [prod_1]", ids)
	}

	populateProductOnSOLine(ctx, line, map[string]any{
		"prod_1": &apiresource.Product{ID: "prod_1", Object: constants.ObjectTypeProduct},
	})
	if line.Product == nil || line.Product.ID != "prod_1" {
		t.Fatalf("line.Product not populated from loaded map, got %+v", line.Product)
	}
}
