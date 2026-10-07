package itemep

import (
	"context"
	"slices"
	"testing"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	_ "github.com/open-mrp/api/services/api-gateway/internal/resourceregistry"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/grpc"
)

// fakeItemCore answers GetItem and fails the test on the include loader's batch read.
type fakeItemCore struct {
	pb.CoreServiceClient
	t          *testing.T
	item       *pb.ItemInfo
	categories []*pb.ItemCategoryInfo
	properties []*pb.PropertyInfo
	err        *apierror.APIError
	request    *pb.GetItemRequest
}

func (f *fakeItemCore) GetItem(_ context.Context, in *pb.GetItemRequest, _ ...grpc.CallOption) (*pb.GetItemResponse, error) {
	f.request = in
	if f.err != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(f.err)
	}
	return &pb.GetItemResponse{Item: f.item, Categories: f.categories, AttributeProperties: f.properties}, nil
}

func (f *fakeItemCore) BatchGetItemCategoriesByIDs(context.Context, *pb.BatchGetItemCategoriesByIDsRequest, ...grpc.CallOption) (*pb.BatchGetItemCategoriesByIDsResponse, error) {
	f.t.Error("the item's category was loaded under item_categories:read instead of read with the item")
	return &pb.BatchGetItemCategoriesByIDsResponse{}, nil
}

func (f *fakeItemCore) BatchGetPropertiesByIDs(context.Context, *pb.BatchGetPropertiesByIDsRequest, ...grpc.CallOption) (*pb.BatchGetPropertiesByIDsResponse, error) {
	f.t.Error("the attributes' properties were loaded under properties:read instead of read with the item")
	return &pb.BatchGetPropertiesByIDsResponse{}, nil
}

func (f *fakeItemCore) BatchGetItemsByIDs(context.Context, *pb.BatchGetItemsByIDsRequest, ...grpc.CallOption) (*pb.BatchGetItemsByIDsResponse, error) {
	f.t.Error("retrieve read the item through the include loader's batch read")
	return &pb.BatchGetItemsByIDsResponse{}, nil
}

func TestGetItem_RefusalFromCoreGetItemStands(t *testing.T) {
	t.Parallel()
	core := &fakeItemCore{t: t, err: apierror.NewAuthorizationError("You do not have access to this resource.")}
	svc := NewItemSvc(&ItemSvcConfig{CoreClient: core})

	item, apiErr := svc.GetItem(context.Background(), &RetrieveItemRequest{ItemID: "itm_1"})

	if apiErr == nil || apiErr.Code != apierror.ErrorCodeInsufficientPerms {
		t.Fatalf("GetItem() = %v, %v; want insufficient_permissions", item, apiErr)
	}
}

func TestGetItem_ReadsTheRecordTheListReads(t *testing.T) {
	t.Parallel()
	core := &fakeItemCore{t: t, item: &pb.ItemInfo{
		Id:           "itm_1",
		Sku:          "SKU-1",
		ItemTypeCode: "material",
		Category:     &pb.ItemCategoryInfo{Id: "itcg_1"},
		UnitCost:     &pb.RateInfo{Id: "rt_cost", Value: "2.5"},
	}}
	svc := NewItemSvc(&ItemSvcConfig{CoreClient: core})
	ctx := resourcekit.WithLoadMeta(context.Background())

	item, apiErr := svc.GetItem(ctx, &RetrieveItemRequest{ItemID: "itm_1"})
	if apiErr != nil {
		t.Fatalf("GetItem() error = %v", apiErr)
	}

	if core.request.GetId() != "itm_1" {
		t.Errorf("GetItem asked for %q, want itm_1", core.request.GetId())
	}
	if got, want := core.request.GetIncludes(), resourceloaders.ItemRecordIncludes; !slices.Equal(got, want) {
		t.Errorf("GetItem includes = %v, want %v", got, want)
	}
	if item.ID != "itm_1" || item.SKU != "SKU-1" || item.Object != constants.ObjectTypeItem {
		t.Errorf("item = %+v", item)
	}
	if item.UnitCost != nil || item.Category != nil {
		t.Errorf("expandable fields must stay null until included: %+v", item)
	}
	if core.request.GetWithCategories() || core.request.GetWithAttributeProperties() {
		t.Errorf("asked core for the item's category or attribute properties though nothing included them: %v", core.request)
	}
	meta := resourcekit.GetLoadMeta(ctx)
	if v, ok := meta.Get(constants.ObjectTypeItem, "itm_1", "unit_cost"); !ok || v == nil {
		t.Error("unit cost not stashed for ?include=unit_cost")
	}
}

// The category and the attributes' properties come back with the item, so the include resolver never loads them under their own permissions.
func TestGetItem_EmbedsTheCategoryAndAttributePropertiesReadWithTheItem(t *testing.T) {
	t.Parallel()
	core := &fakeItemCore{
		t: t,
		item: &pb.ItemInfo{
			Id:         "itm_1",
			Category:   &pb.ItemCategoryInfo{Id: "itcg_1"},
			Attributes: []*pb.ItemAttributeInfo{{Id: "at_1", Value: "Beige", PropertyId: "pp_1"}},
		},
		categories: []*pb.ItemCategoryInfo{{Id: "itcg_1", Name: "Socks", Properties: []*pb.ItemCategoryPropertyInfo{{Id: "pp_1", Name: "Color"}}}},
		properties: []*pb.PropertyInfo{{Id: "pp_1", Name: "Color"}},
	}
	svc := NewItemSvc(&ItemSvcConfig{CoreClient: core})
	ctx := resourcekit.WithLoadMeta(context.Background())
	ctx = resourcekit.WithRequestedIncludes(ctx, []string{"category", "category.properties", "attributes"})

	item, apiErr := svc.GetItem(ctx, &RetrieveItemRequest{ItemID: "itm_1"})
	if apiErr != nil {
		t.Fatalf("GetItem() error = %v", apiErr)
	}
	if !core.request.GetWithCategories() || !core.request.GetWithAttributeProperties() {
		t.Fatalf("core was not asked for the embedded records: %v", core.request)
	}

	tree := resourcekit.NewIncludeTree()
	for _, key := range []string{"category", "category.properties", "attributes"} {
		tree.Add(key)
	}
	if apiErr := resourcekit.ResolveIncludes(ctx, []any{item}, constants.ObjectTypeItem, tree); apiErr != nil {
		t.Fatalf("ResolveIncludes() error = %v", apiErr)
	}
	if item.Category == nil || item.Category.Name != "Socks" {
		t.Fatalf("category = %+v, want the one read with the item", item.Category)
	}
	if item.Category.Properties == nil || len(item.Category.Properties.Data) != 1 || item.Category.Properties.Data[0].Name != "Color" {
		t.Errorf("category properties = %+v", item.Category.Properties)
	}
	if item.Attributes == nil || len(item.Attributes.Data) != 1 || item.Attributes.Data[0].Property == nil || item.Attributes.Data[0].Property.Name != "Color" {
		t.Errorf("attributes = %+v", item.Attributes)
	}
}
