package service

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/open-mrp/api/services/agent-service/internal/agents"
)

const removedEndpointTool = "tool_removed_from_catalog"

func TestValidateAgentConfig_StoredStaleSlugIsTolerated(t *testing.T) {
	t.Parallel()
	known := agents.EndpointTools[0].Slug
	cfg := fmt.Sprintf(`{"endpoint_tool_slugs":[%q,%q],"endpoint_tool_review":{%q:true}}`, known, removedEndpointTool, removedEndpointTool)

	if err := validateAgentConfig(cfg, []byte(cfg)); err != nil {
		t.Fatalf("re-sending a stored stale slug should be accepted, got %v", err)
	}
}

func TestValidateAgentConfig_NewUnknownSlugIsRejected(t *testing.T) {
	t.Parallel()
	known := agents.EndpointTools[0].Slug
	stored := []byte(fmt.Sprintf(`{"endpoint_tool_slugs":[%q]}`, known))

	for _, cfg := range []string{
		fmt.Sprintf(`{"endpoint_tool_slugs":[%q,%q]}`, known, removedEndpointTool),
		fmt.Sprintf(`{"endpoint_tool_review":{%q:true}}`, removedEndpointTool),
	} {
		err := validateAgentConfig(cfg, stored)
		if err == nil {
			t.Fatalf("%s should be rejected", cfg)
		}
		if err.Param != "tools" {
			t.Errorf("%s: param = %v, want tools", cfg, err.Param)
		}
	}
	if err := validateAgentConfig(fmt.Sprintf(`{"endpoint_tool_slugs":[%q]}`, removedEndpointTool), nil); err == nil {
		t.Fatal("an unknown slug on create should be rejected")
	}
}

func TestPruneUnknownEndpointTools(t *testing.T) {
	t.Parallel()
	known := agents.EndpointTools[0].Slug
	in := []byte(fmt.Sprintf(`{"system_prompt":"hi","endpoint_tool_slugs":[%q,%q],"endpoint_tool_review":{%q:true,%q:false}}`, removedEndpointTool, known, removedEndpointTool, known))

	out, err := pruneUnknownEndpointTools(in)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		SystemPrompt       string          `json:"system_prompt"`
		EndpointToolSlugs  []string        `json:"endpoint_tool_slugs"`
		EndpointToolReview map[string]bool `json:"endpoint_tool_review"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if got.SystemPrompt != "hi" {
		t.Errorf("system_prompt = %q, want hi", got.SystemPrompt)
	}
	if len(got.EndpointToolSlugs) != 1 || got.EndpointToolSlugs[0] != known {
		t.Errorf("endpoint_tool_slugs = %v, want [%s]", got.EndpointToolSlugs, known)
	}
	if len(got.EndpointToolReview) != 1 {
		t.Errorf("endpoint_tool_review = %v, want only %s", got.EndpointToolReview, known)
	}
	if _, ok := got.EndpointToolReview[known]; !ok {
		t.Errorf("endpoint_tool_review lost %s", known)
	}
}

func TestPruneUnknownEndpointTools_KeepsWildcard(t *testing.T) {
	t.Parallel()
	out, err := pruneUnknownEndpointTools([]byte(`{"endpoint_tool_slugs":["*"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"endpoint_tool_slugs":["*"]}` {
		t.Errorf("got %s", out)
	}
}

func TestResolveAllowedEndpointTools_IgnoresUnknownSlug(t *testing.T) {
	t.Parallel()
	known := agents.EndpointTools[0].Slug
	got := resolveAllowedEndpointTools([]string{removedEndpointTool, known})
	if len(got) != 1 || !got[known] {
		t.Errorf("got %v, want only %s", got, known)
	}
}
