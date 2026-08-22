package resourceloaders

import (
	"context"
	"testing"

	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/grpc"
)

// stubPickClient answers GetPick from a fixed set of ids and returns not-found for
// everything else. Only GetPick is exercised; the embedded interface satisfies the
// rest and panics if anything else is ever called.
type stubPickClient struct {
	pb.CorePickingServiceClient
	known map[string]*pb.PickInfo
}

func (s stubPickClient) GetPick(_ context.Context, req *pb.GetPickRequest, _ ...grpc.CallOption) (*pb.GetPickResponse, error) {
	if pick, ok := s.known[req.Id]; ok {
		return &pb.GetPickResponse{Pick: pick}, nil
	}
	return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewResourceNotFoundError("Resource not found."))
}

// A pick that has been deleted by the time the include resolves must leave that one
// row's related.pick absent, not fail the whole request. Deleting a shipment deletes
// the pick it was packed from, so without this a single concurrent delete turns any
// list page holding a reference read moments earlier into a 404 in full.
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
