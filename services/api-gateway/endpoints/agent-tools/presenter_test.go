package agenttoolep

import (
	"testing"

	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/augno/api/shared/proto/agent"
)

func TestAvailableToolPresenter(t *testing.T) {
	t.Parallel()
	tool := &pb.AvailableToolInfo{
		Id:               "tdef_01f0c4d04780ace864e6cc3a74",
		DisplayName:      "Search Products",
		Description:      "Search for products by keyword or phrase",
		ConfigSchemaJson: `{}`,
		Category:         "built_in",
	}

	result := availableToolFromProto(tool)
	resourcetest.ValidateResourceStruct(t, "AvailableTool", result)
}
