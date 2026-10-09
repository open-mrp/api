package agents

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
)

// A query whose only overlap is a generic verb ("list") names no operation, so nothing is revealed and the
// agent is told plainly that its toolset lacks it — instead of ten unrelated list_* tools.
func TestHandleSearchAPITools_VerbOnlyMatchIsNoMatch(t *testing.T) {
	t.Parallel()
	runCtx := &domain.HandlerRunContext{AllowedEndpointToolSlugs: grantAllTools()}
	out, err := HandleSearchAPITools(context.Background(), json.RawMessage(`{"query":"list zorblax frobnicators"}`), runCtx)
	if err != nil {
		t.Fatal(err)
	}
	if len(runCtx.RevealedToolSlugs) != 0 {
		t.Errorf("weak matches must not be revealed, got %v", runCtx.RevealedToolSlugs)
	}
	for _, want := range []string{`No operation matches "list zorblax frobnicators"`, "Closest, not made callable", "tell the user"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Count(out, "\n- ") > closestToolLimit {
		t.Errorf("at most %d near misses should be listed:\n%s", closestToolLimit, out)
	}
}

func TestHandleSearchAPITools_NothingAtAll(t *testing.T) {
	t.Parallel()
	runCtx := &domain.HandlerRunContext{AllowedEndpointToolSlugs: grantAllTools()}
	out, _ := HandleSearchAPITools(context.Background(), json.RawMessage(`{"query":"xylophone wizardry"}`), runCtx)
	if !strings.Contains(out, "No operation matches") || strings.Contains(out, "Closest") {
		t.Errorf("unexpected output:\n%s", out)
	}
}

// A subject term matching the slug is a strong match even when the verb differs.
func TestRankEndpointTools_SubjectMatchIsStrong(t *testing.T) {
	t.Parallel()
	strong, _ := rankEndpointTools("create customer", grantTools("create_customer", "list_customers", "create_product"))
	if !hasSlug(strong, "create_customer") || !hasSlug(strong, "list_customers") {
		t.Errorf("customer tools should be strong, got %v", resultSlugs(strong))
	}
	if hasSlug(strong, "create_product") {
		t.Error("create_product shares only the verb and must not be a strong match")
	}
}

// A query of nothing but generic words still finds operations named by them.
func TestRankEndpointTools_AllGenericQueryFallsBack(t *testing.T) {
	t.Parallel()
	strong, _ := rankEndpointTools("list", grantTools("list_customers"))
	if !hasSlug(strong, "list_customers") {
		t.Errorf("an all-generic query should match on those words, got %v", resultSlugs(strong))
	}
}
