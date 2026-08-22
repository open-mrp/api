package agentep

import (
	"testing"

	"github.com/open-mrp/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/open-mrp/api/shared/proto/agent"
)

func TestAgentDefinitionPresenter(t *testing.T) {
	t.Parallel()
	now := "2026-01-01T00:00:00Z"

	def := &pb.AgentDefinitionInfo{
		Id:             "agdf_01abc",
		Name:           "Email Order Agent",
		Slug:           "email_order",
		Description:    new("Processes incoming emails."),
		DefinitionType: "system",
		CategoryCode:   "order_processing",
		TriggerType:    "event",
		IsEditable:     true,
		ConfigJson:     `{"system_prompt":"You are an agent.","model":"claude-sonnet-4"}`,
		Tools: []*pb.AgentDefinitionToolInfo{
			{
				Id:               "agdftl_01abc",
				ToolSlug:         "lookup_customer",
				DisplayName:      "Lookup Customer",
				Description:      "Look up a customer by their email address.",
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
		Description:    new("Processes incoming emails."),
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
