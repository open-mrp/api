package resourceloaders

import (
	"context"
	"testing"

	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/grpc"
)

// stubPickClient answers BatchGetPicksByIDs from a fixed set of ids, omitting unknown ones the way
// core does. The embedded interface panics if anything else is called, so a per-id GetPick fan-out
// fails the test.
type stubPickClient struct {
	pb.CorePickingServiceClient
	known   map[string]*pb.PickInfo
	err     error
	batches *int
}

func (s stubPickClient) BatchGetPicksByIDs(_ context.Context, req *pb.BatchGetPicksByIDsRequest, _ ...grpc.CallOption) (*pb.BatchGetPicksByIDsResponse, error) {
	if s.batches != nil {
		*s.batches++
	}
	if s.err != nil {
		return nil, s.err
	}
	resp := &pb.BatchGetPicksByIDsResponse{}
	for _, id := range req.Ids {
		if pick, ok := s.known[id]; ok {
			resp.Picks = append(resp.Picks, pick)
		}
	}
	return resp, nil
}

// Deleting a shipment deletes the pick it was packed from, so a list page can hold a reference to
// a pick that is gone by the time the include resolves; only that row's related.pick goes null.
func TestLoadPicks_MissingPickIsOmittedNotFatal(t *testing.T) {
	original := corePickingClient
	t.Cleanup(func() { corePickingClient = original })
	corePickingClient = stubPickClient{known: map[string]*pb.PickInfo{
		"pk_alive": {Id: "pk_alive", Number: "PK-1"},
	}}

	out, apiErr := LoadPicks(context.Background(), []string{"pk_alive", "pk_deleted"})
	if apiErr != nil {
		t.Fatalf("LoadPicks returned %v, want the surviving pick and no error", apiErr)
	}
	if _, ok := out["pk_alive"]; !ok {
		t.Error("the pick that still exists must be present")
	}
	if _, ok := out["pk_deleted"]; ok {
		t.Error("the deleted pick must be absent rather than expanded")
	}
}

func TestLoadPicks_OneRoundTripForAPage(t *testing.T) {
	original := corePickingClient
	t.Cleanup(func() { corePickingClient = original })
	batches := 0
	corePickingClient = stubPickClient{batches: &batches, known: map[string]*pb.PickInfo{
		"pk_1": {Id: "pk_1", Number: "PK-1"},
		"pk_2": {Id: "pk_2", Number: "PK-2"},
		"pk_3": {Id: "pk_3", Number: "PK-3"},
	}}

	out, apiErr := LoadPicks(context.Background(), []string{"pk_1", "pk_2", "pk_3"})
	if apiErr != nil {
		t.Fatalf("LoadPicks: %v", apiErr)
	}
	if len(out) != 3 {
		t.Errorf("got %d picks, want 3", len(out))
	}
	if batches != 1 {
		t.Errorf("made %d batch calls, want 1", batches)
	}
}

// An actor without pick read access still gets the rest of the page, with related.pick null.
func TestLoadPicks_UnauthorizedOmitsAll(t *testing.T) {
	original := corePickingClient
	t.Cleanup(func() { corePickingClient = original })
	corePickingClient = stubPickClient{err: contracts.ConvertAPIErrorToGRPC(apierror.NewAuthorizationError("No access."))}

	out, apiErr := LoadPicks(context.Background(), []string{"pk_1"})
	if apiErr != nil {
		t.Fatalf("LoadPicks returned %v, want an empty result", apiErr)
	}
	if len(out) != 0 {
		t.Errorf("got %d picks, want none", len(out))
	}
}
