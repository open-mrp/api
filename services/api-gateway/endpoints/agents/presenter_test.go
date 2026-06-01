package agentep

import (
	"testing"

	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/augno/api/shared/proto/agent"
)

func TestAgentDefinitionPresenter(t *testing.T) {
	t.Parallel()
	now := "2026-01-01T00:00:00Z"

	def := &pb.AgentDefinitionInfo{
		Id:             "agdf_01abc",
		Name:           "Email Order Agent",
		Slug:           "email_order",
		Description:    "Processes incoming emails.",
		DefinitionType: "system",
		CategoryCode:   "order_processing",
		TriggerType:    "event",
		IsEditable:     true,
		ConfigJson:     `{"system_prompt":"You are an agent.","model":"claude-sonnet-4"}`,
		Tools: []*pb.AgentDefinitionToolInfo{
			{
				Id:               "agdftl_01abc",
				ToolId:           "tdef_01f0c4d04780ace864e6cc3a74",
				DisplayName:      "Search Products",
				Description:      "Search for products by keyword or phrase",
				ConfigSchemaJson: `{}`,
				Category:         "built_in",
				ConfigJson:       `{}`,
				SortOrder:        1,
				RequireReview:    true,
			},
		},
		RoleId: "rl_01abc",
		AccountStatus: &pb.AgentAccountStatusInfo{
			StatusCode: "active",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	result := AgentDefinitionPresenter(def)
	resourcetest.ValidateExpandableStubs(t, "AgentDefinition", result)
}

func TestAgentDefinitionPresenter_NoRole(t *testing.T) {
	t.Parallel()
	now := "2026-01-01T00:00:00Z"

	def := &pb.AgentDefinitionInfo{
		Id:             "agdf_01abc",
		Name:           "Email Order Agent",
		Slug:           "email_order",
		Description:    "Processes incoming emails.",
		DefinitionType: "system",
		CategoryCode:   "order_processing",
		TriggerType:    "event",
		IsEditable:     true,
		AccountStatus: &pb.AgentAccountStatusInfo{
			StatusCode: "active",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	result := AgentDefinitionPresenter(def)
	resourcetest.ValidateExpandableStubs(t, "AgentDefinition(NoRole)", result)
}

func TestAgentTokenUsagePresenter(t *testing.T) {
	t.Parallel()
	now := "2026-01-01T00:00:00Z"

	usage := &pb.AgentTokenUsageInfo{
		Id:           "agtu_01abc",
		Date:         "2026-01-01",
		InputTokens:  100,
		OutputTokens: 200,
		TotalCost:    0.01,
		RunCount:     1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	result := AgentTokenUsagePresenter(usage)
	resourcetest.ValidateResourceStruct(t, "AgentTokenUsage", result)
}
