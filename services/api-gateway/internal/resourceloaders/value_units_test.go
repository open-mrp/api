package resourceloaders

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/open-mrp/apikit/include"
	"google.golang.org/grpc"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	pb "github.com/open-mrp/api/shared/proto/core"
)

// stubUnitClient answers BatchGetUnitsByIDs and records each batch it was asked for.
type stubUnitClient struct {
	pb.CoreServiceClient
	batches *[][]string
}

func (s stubUnitClient) BatchGetUnitsByIDs(_ context.Context, req *pb.BatchGetUnitsByIDsRequest, _ ...grpc.CallOption) (*pb.BatchGetUnitsByIDsResponse, error) {
	ids := slices.Clone(req.Ids)
	slices.Sort(ids)
	*s.batches = append(*s.batches, ids)
	units := make([]*pb.UnitInfo, 0, len(req.Ids))
	for _, id := range req.Ids {
		units = append(units, &pb.UnitInfo{Id: id, Name: "Unit " + id, Abbreviation: id, Type: "mass"})
	}
	return &pb.BatchGetUnitsByIDsResponse{Units: units}, nil
}

type valueHolder struct {
	Price    *apiresource.UnitPrice
	Lines    *apiresource.List[apiresource.Quantity]
	Anything any
	Named    map[string]*apiresource.Rate
}

func TestFillUnits_LoadsEveryStubInOneBatchAndReusesIt(t *testing.T) {
	var batches [][]string
	original := coreClient
	t.Cleanup(func() { coreClient = original })
	coreClient = stubUnitClient{batches: &batches}

	ctx := include.WithLoadMeta(context.Background())
	v := &valueHolder{
		Price:    apiresource.NewUnitPrice("2.50", "un_kg"),
		Lines:    apiresource.NewList([]apiresource.Quantity{*apiresource.NewQuantity("3", "un_kg"), *apiresource.NewQuantity("4", "un_lb")}, apiresource.PageInfo{}),
		Anything: apiresource.NewQuantity("5", "un_g"),
		Named:    map[string]*apiresource.Rate{"burn": apiresource.NewRate("1", "un_g", "un_hr")},
	}
	if apiErr := FillUnits(ctx, v); apiErr != nil {
		t.Fatalf("FillUnits: %v", apiErr)
	}
	if want := [][]string{{"un_g", "un_hr", "un_kg", "un_lb"}}; !reflect.DeepEqual(batches, want) {
		t.Fatalf("batches = %v, want %v", batches, want)
	}
	for name, u := range map[string]*apiresource.Unit{
		"price":     v.Price.PerUnit,
		"line":      v.Lines.Data[1].Unit,
		"interface": v.Anything.(*apiresource.Quantity).Unit,
		"map":       v.Named["burn"].PerUnit,
	} {
		if apiresource.IsUnitStub(u) || u.Name == "" {
			t.Errorf("%s unit was not filled: %+v", name, u)
		}
	}

	again := apiresource.NewQuantity("6", "un_kg")
	if apiErr := FillUnits(ctx, again); apiErr != nil {
		t.Fatalf("FillUnits: %v", apiErr)
	}
	if len(batches) != 1 || again.Unit.Name != "Unit un_kg" {
		t.Errorf("a unit loaded earlier in the request was fetched again: %v", batches)
	}
}

func TestMayHoldUnit(t *testing.T) {
	t.Parallel()

	for typ, want := range map[reflect.Type]bool{
		reflect.TypeFor[*apiresource.Item]():                   true,
		reflect.TypeFor[*apiresource.List[apiresource.Item]](): true,
		reflect.TypeFor[*apiresource.Money]():                  false,
		reflect.TypeFor[*apiresource.Healthcheck]():            false,
	} {
		if got := MayHoldUnit(typ); got != want {
			t.Errorf("MayHoldUnit(%v) = %v, want %v", typ, got, want)
		}
	}
}

func TestUnitStubIDsAndAttachUnits(t *testing.T) {
	t.Parallel()

	q := apiresource.NewQuantity("1", "un_kg")
	if ids := UnitStubIDs(q); !reflect.DeepEqual(ids, []string{"un_kg"}) {
		t.Fatalf("UnitStubIDs = %v", ids)
	}
	AttachUnits(q, map[string]any{"un_kg": &apiresource.Unit{ID: "un_kg", Object: "unit", Name: "Kilogram"}})
	if q.Unit.Name != "Kilogram" {
		t.Errorf("unit not attached: %+v", q.Unit)
	}
	if ids := UnitStubIDs(q); len(ids) != 0 {
		t.Errorf("a filled unit is still reported as a stub: %v", ids)
	}
}
