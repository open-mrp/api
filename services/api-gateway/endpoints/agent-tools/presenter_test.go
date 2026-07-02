package agenttoolep

import (
	"testing"

	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/augno/api/shared/proto/agent"
)

func TestAvailableToolPresenter(t *testing.T) {
	t.Parallel()
	tool := &pb.AvailableToolInfo{
		Slug:             "lookup_customer",
		DisplayName:      "Lookup Customer",
		Description:      "Look up a customer by their email address.",
		ConfigSchemaJson: `{}`,
		Category:         "built_in",
		Mutating:         true,
	}

	result := availableToolFromProto(tool)
	resourcetest.ValidateResourceStruct(t, "AvailableTool", result)
}
