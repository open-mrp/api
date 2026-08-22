package resourceloaders

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/api-gateway/pkg/resource/resourcetest"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// The *FromProto builders must produce gated resources: expandable nested resources (item, product_line) stay nil so they are only ever populated when the caller explicitly requests them via ?include=. The full XxxPresenter functions, by contrast, embed those nested resources and are reserved for the Excel export path. These tests lock the gated contract in place.

func TestProductFromProto_LeavesExpandablesNil(t *testing.T) {
	t.Parallel()
	now := timestamppb.Now()
	productLineID := "prl_01abc"
	p := &pb.ProductFullInfo{
		Id:              "prod_01abc",
		ProductTypeCode: string(constants.ProductTypeCodeSale),
		ItemId:          "item_01abc",
		ProductLineId:   &productLineID,
		IsPortalReady:   true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	got := ProductFromProto(p)

	// Core fields are populated.
	if got.ID != "prod_01abc" {
		t.Errorf("ID = %q, want prod_01abc", got.ID)
	}
	if got.Type == "" {
		t.Errorf("Type is empty — gated builder must still set core enum fields")
	}
	if got.PortalVisibility != constants.CustomerPortalVisibilityVisible {
		t.Errorf("PortalVisibility = %q, want visible", got.PortalVisibility)
	}
	// Expandables stay nil even though item_id / product_line_id are present.
	if got.Item != nil {
		t.Errorf("Item must be nil until requested via ?include=item, got %+v", got.Item)
	}
	if got.ProductLine != nil {
		t.Errorf("ProductLine must be nil until requested via ?include=product_line, got %+v", got.ProductLine)
	}
	resourcetest.AssertExpandablesNil(t, "Product", got)
}

func TestMaterialFromProto_LeavesExpandablesNil(t *testing.T) {
	t.Parallel()
	now := timestamppb.Now()
	m := &pb.MaterialInfo{
		Id:        "mat_01abc",
		ItemId:    "item_01abc",
		CreatedAt: now,
		UpdatedAt: now,
	}

	got := MaterialFromProto(m)

	if got.ID != "mat_01abc" {
		t.Errorf("ID = %q, want mat_01abc", got.ID)
	}
	if got.Item != nil {
		t.Errorf("Item must be nil until requested via ?include=item, got %+v", got.Item)
	}
	resourcetest.AssertExpandablesNil(t, "Material", got)
}

func TestPartFromProto_LeavesExpandablesNil(t *testing.T) {
	t.Parallel()
	now := timestamppb.Now()
	p := &pb.PartInfo{
		Id:        "part_01abc",
		ItemId:    "item_01abc",
		CreatedAt: now,
		UpdatedAt: now,
	}

	got := PartFromProto(p)

	if got.ID != "part_01abc" {
		t.Errorf("ID = %q, want part_01abc", got.ID)
	}
	if got.Item != nil {
		t.Errorf("Item must be nil until requested via ?include=item, got %+v", got.Item)
	}
	resourcetest.AssertExpandablesNil(t, "Part", got)
}

// The Stash helpers must record the ids the include resolver needs so that ?include= still works after the gated build (otherwise we'd trade a leak for a permanently-missing include). Pairs with the gated builders above.

func TestStashProductMeta_RecordsIncludeIDs(t *testing.T) {
	t.Parallel()
	ctx := resourcekit.WithLoadMeta(context.Background())
	productLineID := "prl_01abc"
	StashProductMeta(ctx, &pb.ProductFullInfo{
		Id:            "prod_01abc",
		ItemId:        "item_01abc",
		ProductLineId: &productLineID,
	})

	meta := resourcekit.GetLoadMeta(ctx)
	if v, _ := meta.GetString(constants.ObjectTypeProduct, "prod_01abc", "item_id"); v != "item_01abc" {
		t.Errorf("item_id meta = %q, want item_01abc", v)
	}
	if v, _ := meta.GetString(constants.ObjectTypeProduct, "prod_01abc", "product_line_id"); v != "prl_01abc" {
		t.Errorf("product_line_id meta = %q, want prl_01abc", v)
	}
}

func TestStashMaterialMeta_RecordsItemID(t *testing.T) {
	t.Parallel()
	ctx := resourcekit.WithLoadMeta(context.Background())
	StashMaterialMeta(ctx, &pb.MaterialInfo{Id: "mat_01abc", ItemId: "item_01abc"})

	if v, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeMaterial, "mat_01abc", "item_id"); v != "item_01abc" {
		t.Errorf("item_id meta = %q, want item_01abc", v)
	}
}

func TestStashPartMeta_RecordsItemID(t *testing.T) {
	t.Parallel()
	ctx := resourcekit.WithLoadMeta(context.Background())
	StashPartMeta(ctx, &pb.PartInfo{Id: "part_01abc", ItemId: "item_01abc"})

	if v, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypePart, "part_01abc", "item_id"); v != "item_01abc" {
		t.Errorf("item_id meta = %q, want item_01abc", v)
	}
}
