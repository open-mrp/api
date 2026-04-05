package sandboxep

import (
	"testing"

	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSandboxPresenter(t *testing.T) {
	t.Parallel()
	ownerID := "ac_01abc"
	ownerName := "Acme Inc."

	sandbox := &pb.SandboxInfo{
		Id:               "sbac_01abc",
		Name:             "Test Sandbox",
		CreatedAt:        timestamppb.Now(),
		UpdatedAt:        timestamppb.Now(),
		OwnerAccountId:   &ownerID,
		OwnerAccountName: &ownerName,
	}

	result := SandboxPresenter(sandbox)
	resourcetest.ValidateResourceStruct(t, "Sandbox", result)
}
