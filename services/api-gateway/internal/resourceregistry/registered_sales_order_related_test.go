package resourceregistry

import (
	"context"
	"testing"
	"time"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

// The sales order's related records (pick / production_run / shipments) carry only ids on SalesOrderInfo; their number and status must be loaded from the owning service via the include resolver. These tests lock the extract/populate wiring that turns a loaded resource into a Record with number + status.

func TestRelatedPick_PopulatesNumberAndStatus(t *testing.T) {
	ctx := resourcekit.WithLoadMeta(context.Background())
	resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeSalesOrder, "or_1", "related_pick_id", "pk_1")
	so := &apiresource.SalesOrder{ID: "or_1"}

	if ids := extractPickIDFromSORelated(ctx, so); len(ids) != 1 || ids[0] != "pk_1" {
		t.Fatalf("extract = %v, want [pk_1]", ids)
	}

	finished := time.Now()
	populatePickOnSORelated(ctx, so, map[string]any{
		"pk_1": &apiresource.Pick{ID: "pk_1", Number: "PK-001", FinishedAt: &finished},
	})

	if so.Related == nil || so.Related.Pick == nil {
		t.Fatal("related.pick not populated")
	}
	rec := so.Related.Pick
	if rec.Number == nil || *rec.Number != "PK-001" {
		t.Errorf("pick number = %v, want PK-001", rec.Number)
	}
	if rec.Status == nil || *rec.Status != "closed" {
		t.Errorf("pick status = %v, want closed (FinishedAt set)", rec.Status)
	}
}

func TestRelatedPick_OpenWhenNotFinished(t *testing.T) {
	ctx := resourcekit.WithLoadMeta(context.Background())
	resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeSalesOrder, "or_1", "related_pick_id", "pk_1")
	so := &apiresource.SalesOrder{ID: "or_1"}

	populatePickOnSORelated(ctx, so, map[string]any{
		"pk_1": &apiresource.Pick{ID: "pk_1", Number: "PK-001"},
	})

	if so.Related.Pick.Status == nil || *so.Related.Pick.Status != "open" {
		t.Errorf("pick status = %v, want open (no FinishedAt)", so.Related.Pick.Status)
	}
}

func TestRelatedProductionRun_PopulatesNumberAndStatus(t *testing.T) {
	ctx := resourcekit.WithLoadMeta(context.Background())
	resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeSalesOrder, "or_1", "related_production_run_id", "pr_1")
	so := &apiresource.SalesOrder{ID: "or_1"}

	completed := time.Now()
	populateProductionRunOnSORelated(ctx, so, map[string]any{
		"pr_1": &apiresource.ProductionRun{ID: "pr_1", Number: "1", CompletedAt: &completed},
	})

	rec := so.Related.ProductionRun
	if rec == nil || rec.Number == nil || *rec.Number != "1" {
		t.Fatalf("production_run number not populated, got %+v", rec)
	}
	if rec.Status == nil || *rec.Status != "closed" {
		t.Errorf("production_run status = %v, want closed (CompletedAt set)", rec.Status)
	}
}

func TestRelatedShipments_PopulateNumberAndStatus(t *testing.T) {
	ctx := resourcekit.WithLoadMeta(context.Background())
	resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeSalesOrder, "or_1", "related_shipment_ids", []string{"sh_1", "sh_2"})
	so := &apiresource.SalesOrder{ID: "or_1"}

	if ids := extractShipmentIDsFromSORelated(ctx, so); len(ids) != 2 {
		t.Fatalf("extract = %v, want 2 ids", ids)
	}

	populateShipmentsOnSORelated(ctx, so, map[string]any{
		"sh_1": &apiresource.Shipment{ID: "sh_1", Number: "SH-001", Status: constants.ShipmentStatus("fulfilled")},
		"sh_2": &apiresource.Shipment{ID: "sh_2", Number: "SH-002", Status: constants.ShipmentStatus("pending")},
	})

	if so.Related == nil || so.Related.Shipments == nil {
		t.Fatal("related.shipments not populated")
	}
	data := so.Related.Shipments.Data
	if len(data) != 2 {
		t.Fatalf("want 2 shipment records, got %d", len(data))
	}
	if data[0].Number == nil || *data[0].Number != "SH-001" || data[0].Status == nil || *data[0].Status != "fulfilled" {
		t.Errorf("shipment[0] = number %v status %v, want SH-001/fulfilled", data[0].Number, data[0].Status)
	}
}

// When the order references no related records (nothing stashed), the related group must stay nil so it serializes to null rather than an empty object.
func TestRelated_NilWhenNothingStashed(t *testing.T) {
	ctx := resourcekit.WithLoadMeta(context.Background())
	so := &apiresource.SalesOrder{ID: "or_1"}

	populatePickOnSORelated(ctx, so, nil)
	populateProductionRunOnSORelated(ctx, so, nil)
	populateShipmentsOnSORelated(ctx, so, nil)

	if so.Related != nil {
		t.Errorf("related must stay nil when no related records exist, got %+v", so.Related)
	}
}
