package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
)

// HandleDescribeAPIOperation returns the generated contract of one of the agent's granted operations — route,
// permissions, parameters with their location, include values, and the input schema — so the agent stops
// guessing parameter names and include keys. It also makes the operation callable, like a search hit.
func HandleDescribeAPIOperation(_ context.Context, input json.RawMessage, runCtx *domain.HandlerRunContext) (string, error) {
	var params struct {
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid describe_api_operation input: %w", err)
	}
	slug := strings.TrimSpace(params.Slug)
	d, ok := LookupEndpointTool(slug)
	if !ok || !runCtx.AllowedEndpointToolSlugs[slug] {
		return "", fmt.Errorf("describe_api_operation: %q is not one of this agent's API operations; use search_api_tools to find the right slug", slug)
	}
	if runCtx.RevealedToolSlugs == nil {
		runCtx.RevealedToolSlugs = map[string]bool{}
	}
	runCtx.RevealedToolSlugs[slug] = true
	return describeEndpointTool(d), nil
}

func describeEndpointTool(d EndpointToolDescriptor) string {
	var schema struct {
		Properties map[string]struct {
			Type        any    `json:"type"`
			Description string `json:"description"`
			Enum        []any  `json:"enum"`
			Items       *struct {
				Enum []any `json:"enum"`
			} `json:"items"`
		} `json:"properties"`
		Required []string `json:"required"`
	}
	_ = json.Unmarshal([]byte(d.InputSchema), &schema)
	required := make(map[string]bool, len(schema.Required))
	for _, r := range schema.Required {
		required[r] = true
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s — %s\n%s %s\n", d.Slug, d.DisplayName, d.Method, d.RouteTemplate)
	if d.Description != "" {
		fmt.Fprintf(&b, "\n%s\n", strings.TrimSpace(d.Description))
	}
	b.WriteString("\n")
	if d.Mutating() {
		b.WriteString("Changes data: yes\n")
	} else {
		b.WriteString("Changes data: no\n")
	}
	if len(d.RequiredPermissions) > 0 {
		fmt.Fprintf(&b, "Required permissions: %s\n", strings.Join(d.RequiredPermissions, ", "))
	}
	if d.RequiredRoleType != "" {
		fmt.Fprintf(&b, "Required role type: %s\n", d.RequiredRoleType)
	}

	if len(d.Params) > 0 {
		b.WriteString("\nParameters:\n")
		params := append([]EndpointToolParam(nil), d.Params...)
		sort.SliceStable(params, func(i, j int) bool { return required[params[i].Name] && !required[params[j].Name] })
		for _, p := range params {
			prop := schema.Properties[p.Name]
			req := ""
			if required[p.Name] {
				req = ", required"
			}
			fmt.Fprintf(&b, "- %s (%s%s)", p.Name, p.In, req)
			if desc := firstLine(prop.Description); desc != "" {
				fmt.Fprintf(&b, ": %s", desc)
			}
			if len(prop.Enum) > 0 {
				fmt.Fprintf(&b, " One of: %s.", joinValues(prop.Enum))
			}
			b.WriteString("\n")
		}
	}
	if inc, ok := schema.Properties["include"]; ok && inc.Items != nil && len(inc.Items.Enum) > 0 {
		fmt.Fprintf(&b, "\nValid include values: %s\n", joinValues(inc.Items.Enum))
	}
	fmt.Fprintf(&b, "\nInput schema:\n%s\n", d.InputSchema)
	return b.String()
}

func joinValues(vals []any) string {
	out := make([]string, len(vals))
	for i, v := range vals {
		out[i] = fmt.Sprint(v)
	}
	return strings.Join(out, ", ")
}
