package itemep

import (
	"context"
	"slices"
	"testing"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
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
	t       *testing.T
	item    *pb.ItemInfo
	err     *apierror.APIError
	request *pb.GetItemRequest
}

func (f *fakeItemCore) GetItem(_ context.Context, in *pb.GetItemRequest, _ ...grpc.CallOption) (*pb.GetItemResponse, error) {
	f.request = in
	if f.err != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(f.err)
	}
	return &pb.GetItemResponse{Item: f.item}, nil
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
	meta := resourcekit.GetLoadMeta(ctx)
	if id, _ := meta.GetString(constants.ObjectTypeItem, "itm_1", "item_category_id"); id != "itcg_1" {
		t.Errorf("category id stashed = %q, want itcg_1", id)
	}
	if v, ok := meta.Get(constants.ObjectTypeItem, "itm_1", "unit_cost"); !ok || v == nil {
		t.Error("unit cost not stashed for ?include=unit_cost")
	}
}
