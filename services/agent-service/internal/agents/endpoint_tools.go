package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"

	"github.com/augno/api/services/agent-service/internal/domain"
	"github.com/augno/api/shared/fuzzy"
)

// EndpointToolParamLocation identifies where a tool input value goes when building the gateway HTTP request.
type EndpointToolParamLocation string

const (
	EndpointToolParamPath  EndpointToolParamLocation = "path"
	EndpointToolParamQuery EndpointToolParamLocation = "query"
	EndpointToolParamBody  EndpointToolParamLocation = "body"
)

// EndpointToolParam describes one input field of a generated endpoint-tool and where it belongs in the outgoing request.
type EndpointToolParam struct {
	Name  string
	In    EndpointToolParamLocation
	Array bool
}

// EndpointToolDescriptor is the generated description of an api-gateway endpoint exposed as an agent tool. The catalog of these (EndpointTools) is code-generated from endpoints that are public or flagged AgentTool=true; see `make gen-agent-tools`. These tools are offered to every agent in addition to its explicitly-linked hand-crafted tools — adding one needs no database migration, only a regenerate.
type EndpointToolDescriptor struct {
	Slug          string
	DisplayName   string
	Description   string
	Method        string
	RouteTemplate string
	// InputSchema is the self-contained JSON Schema shown to the LLM.
	InputSchema string
	Params      []EndpointToolParam
	// RequiredPermissions are the "<domain>:<action>" permissions this operation needs (from the endpoint's declaration). Surfaced in the tool-selection UI.
	RequiredPermissions []string
	// RequiredRoleType, when non-empty, is the role type the caller must have (e.g. "admin"). Surfaced in the tool-selection UI.
	RequiredRoleType string
	// Group is the display group for the tool-selection UI (e.g. "Customers").
	Group string
	// ReadOnly mirrors the endpoint's own declaration that it computes an answer without changing anything despite not being a GET (a quote, a preview, an analytics query). Generated from the endpoint, never inferred here.
	ReadOnly bool
}

// Mutating reports whether invoking this tool changes server state.
//
// The method is the default answer, because a POST or PUT that writes nothing is the exception rather than the rule — but it is only the default. Quotes, previews and analytics take a request body and so cannot be GETs, and reporting those as mutating puts them in front of a merchant deciding which tools an agent may run unsupervised, next to the ones that actually create orders. The endpoint says which it is; this reads that.
func (d EndpointToolDescriptor) Mutating() bool {
	return d.Method != "GET" && !d.ReadOnly
}

// SearchAPIToolsSlug is the meta-tool agents use to discover endpoint-tools on demand (progressive disclosure), instead of having all of them injected up front.
const SearchAPIToolsSlug = "search_api_tools"

// SearchAPIToolsDescription / SearchAPIToolsInputSchema define the meta-tool as shown to the LLM. The runner adds this tool to a run only when the agent has been granted at least one endpoint-tool.
const (
	SearchAPIToolsDescription = "Search the catalog of available API operations and make the matching ones callable. Describe what you need in plain language (e.g. \"list customers\", \"create a sales order\"); the matching tools are returned and become available to call directly on the next step. Only operations this agent has been granted are searchable."
	SearchAPIToolsInputSchema = `{"type":"object","properties":{"query":{"type":"string","description":"Plain-language description of the operation you need, e.g. \"list open sales orders\" or \"create a customer\"."}},"required":["query"]}`
)

// searchResultLimit caps how many tools a single search reveals, keeping context bounded.
const searchResultLimit = 10

var endpointToolIndex = func() map[string]EndpointToolDescriptor {
	m := make(map[string]EndpointToolDescriptor, len(EndpointTools))
	for _, d := range EndpointTools {
		m[d.Slug] = d
	}
	return m
}()

// LookupEndpointTool returns the catalog descriptor for a slug.
func LookupEndpointTool(slug string) (EndpointToolDescriptor, bool) {
	d, ok := endpointToolIndex[slug]
	return d, ok
}

// RegisterEndpointTools registers a handler for every generated endpoint-tool plus the search_api_tools meta-tool. Each endpoint handler maps the LLM's flat input object onto an HTTP call into the api-gateway via the run's GatewayClient.
func RegisterEndpointTools(registry *ToolHandlerRegistry) {
	for _, d := range EndpointTools {
		registry.Register(d.Slug, endpointToolHandler(d))
	}
	registry.Register(SearchAPIToolsSlug, HandleSearchAPITools)
}

// AllowedToolGroups returns the distinct display groups of the endpoint-tools in the allowed set, sorted, so an agent can be told in plain language what domains it can act on.
func AllowedToolGroups(allowed map[string]bool) []string {
	seen := make(map[string]bool)
	var groups []string
	for _, d := range EndpointTools {
		if !allowed[d.Slug] || d.Group == "" || seen[d.Group] {
			continue
		}
		seen[d.Group] = true
		groups = append(groups, d.Group)
	}
	sort.Strings(groups)
	return groups
}

// SearchEndpointTools returns up to limit catalog tools matching query, restricted to the allowed set. Matching is simple term overlap against each tool's slug and description.
func SearchEndpointTools(query string, allowed map[string]bool, limit int) []EndpointToolDescriptor {
	terms := tokenize(query)
	type scored struct {
		d       EndpointToolDescriptor
		score   int
		unnamed int
	}
	var matches []scored
	for _, d := range EndpointTools {
		if !allowed[d.Slug] {
			continue
		}
		score := scoreTool(d, terms)
		// With no query terms, surface allowed tools alphabetically (score 0). With terms, a tool that matches none is dropped.
		if len(terms) > 0 && score == 0 {
			continue
		}
		matches = append(matches, scored{d: d, score: score, unnamed: unnamedSegments(d, terms)})
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		if matches[i].unnamed != matches[j].unnamed {
			return matches[i].unnamed < matches[j].unnamed
		}
		return matches[i].d.Slug < matches[j].d.Slug
	})
	out := make([]EndpointToolDescriptor, 0, limit)
	for _, m := range matches {
		if len(out) >= limit {
			break
		}
		out = append(out, m.d)
	}
	return out
}

// scoreTool ranks a tool against the query terms, weighting WHERE each term matches: a match on the slug or display name — what the operation actually IS — counts far more than an incidental mention in the prose description or route. Without this weighting a search like "create a customer" ties create_customer with any endpoint whose description merely mentions customers (e.g. create_carrier), and the alphabetical tie-break then surfaces the wrong tool. A term matching a whole slug segment
// (the strongest intent signal) outscores a mere substring (e.g. singular/plural).
//
// When a term matches nothing exactly it falls back to a fuzzy (typo-tolerant) match against the slug segments and display-name words, so a misspelled query like "create custmer" or "updaet customer" still finds the right tool. Fuzzy matches always score below their exact equivalents, so a real match wins whenever one exists. Returns the summed best-field score across all terms; 0 means no term matched anywhere, even fuzzily.
func scoreTool(d EndpointToolDescriptor, terms []string) int {
	if len(terms) == 0 {
		return 0
	}
	slug := strings.ToLower(d.Slug)
	slugSegments := strings.Split(slug, "_")
	name := strings.ToLower(d.DisplayName)
	nameWords := strings.Fields(name)
	desc := strings.ToLower(d.Description)
	route := strings.ToLower(d.RouteTemplate)

	total := 0
	for _, t := range terms {
		best := 0
		switch {
		case containsSegment(slugSegments, t):
			best = 10
		case strings.Contains(slug, t):
			best = 8
		case strings.Contains(name, t):
			best = 6
		case strings.Contains(desc, t), strings.Contains(route, t):
			best = 1
		}
		// Typo tolerance: only worth trying when the term didn't already strongly match an identifier
		// (best < the slug-substring weight). A near-miss of a slug segment is treated as a strong-ish signal; of a name word, weaker.
		if best < 8 {
			if fuzzy.AnyTypo(t, slugSegments) {
				best = max(best, 7)
			} else if fuzzy.AnyTypo(t, nameWords) {
				best = max(best, 4)
			}
		}
		total += best
	}
	return total
}

func containsSegment(segments []string, term string) bool {
	return slices.Contains(segments, term)
}

// unnamedSegments counts the slug segments the query never mentions, breaking ties between tools that scored the same because the caller happened to name every term they have in common. "create a sales order" scores create_sales_order and create_production_run_from_sales_order identically — each matches all three terms on a slug segment — but the latter carries three segments nobody asked for, and is a different operation that merely starts from a sales order. Fewer leftovers means the tool IS what was asked for rather than something containing it, so the plain alphabetical tie-break must not be what decides this.
func unnamedSegments(d EndpointToolDescriptor, terms []string) int {
	count := 0
	for seg := range strings.SplitSeq(strings.ToLower(d.Slug), "_") {
		named := slices.ContainsFunc(terms, func(t string) bool {
			return strings.Contains(seg, t) || fuzzy.AnyTypo(t, []string{seg})
		})
		if !named {
			count++
		}
	}
	return count
}

func tokenize(s string) []string {
	var terms []string
	for _, f := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}) {
		if len(f) > 1 { // drop single-char noise
			terms = append(terms, f)
		}
	}
	return terms
}

// HandleSearchAPITools searches the agent's granted endpoint-tools and records the matches as revealed so the runner makes them callable.
func HandleSearchAPITools(_ context.Context, input json.RawMessage, runCtx *domain.HandlerRunContext) (string, error) {
	var params struct {
		Query string `json:"query"`
	}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &params); err != nil {
			return "", fmt.Errorf("invalid search_api_tools input: %w", err)
		}
	}

	matches := SearchEndpointTools(params.Query, runCtx.AllowedEndpointToolSlugs, searchResultLimit)
	if len(matches) == 0 {
		return "No API tools matched your query.", nil
	}

	if runCtx.RevealedToolSlugs == nil {
		runCtx.RevealedToolSlugs = make(map[string]bool, len(matches))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Found %d API tool(s), now available to call directly:\n", len(matches))
	for _, m := range matches {
		runCtx.RevealedToolSlugs[m.Slug] = true
		fmt.Fprintf(&b, "- %s — %s [%s %s]\n", m.Slug, firstLine(m.Description), m.Method, m.RouteTemplate)
	}
	return b.String(), nil
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return strings.TrimSpace(line)
}

// RevealedSlugsForQueries replays a run's earlier search_api_tools queries against the supplied grant and returns the union of endpoint-tool slugs they surface. search_api_tools' reveals live only on the per-turn run context, so the runner uses this on resume to restore, from the recorded queries, the set an earlier turn made callable — without it a follow-up message that calls an already-discovered tool is denied until the model re-searches. Scoping the replay to the live grant re-checks permission for free: a query that previously surfaced a since-revoked tool no longer returns it, so the tool stays hidden and the execution guard still denies it.
func RevealedSlugsForQueries(queries []string, allowed map[string]bool) map[string]bool {
	revealed := make(map[string]bool)
	if len(allowed) == 0 {
		return revealed
	}
	for _, q := range queries {
		for _, m := range SearchEndpointTools(q, allowed, searchResultLimit) {
			revealed[m.Slug] = true
		}
	}
	return revealed
}

func endpointToolHandler(desc EndpointToolDescriptor) domain.ToolHandlerFunc {
	return func(ctx context.Context, input json.RawMessage, runCtx *domain.HandlerRunContext) (string, error) {
		if runCtx.GatewayClient == nil {
			return "", fmt.Errorf("%s: gateway client not configured", desc.Slug)
		}

		var raw map[string]json.RawMessage
		if len(input) > 0 {
			if err := json.Unmarshal(input, &raw); err != nil {
				return "", fmt.Errorf("invalid input for %s: %w", desc.Slug, err)
			}
		}

		path := desc.RouteTemplate
		query := url.Values{}
		body := map[string]json.RawMessage{}

		for _, p := range desc.Params {
			val, ok := raw[p.Name]
			if !ok {
				continue
			}
			switch p.In {
			case EndpointToolParamPath:
				s, err := jsonScalarToString(val)
				if err != nil {
					return "", fmt.Errorf("%s: path param %q: %w", desc.Slug, p.Name, err)
				}
				path = strings.ReplaceAll(path, "{"+p.Name+"}", url.PathEscape(s))
			case EndpointToolParamQuery:
				if err := appendQueryValue(query, p.Name, val, p.Array); err != nil {
					return "", fmt.Errorf("%s: query param %q: %w", desc.Slug, p.Name, err)
				}
			case EndpointToolParamBody:
				body[p.Name] = val
			}
		}

		var bodyJSON json.RawMessage
		if len(body) > 0 {
			b, err := json.Marshal(body)
			if err != nil {
				return "", fmt.Errorf("%s: encoding body: %w", desc.Slug, err)
			}
			bodyJSON = b
		}

		// For mutating calls, derive a deterministic idempotency key from the run and tool-use IDs so a re-delivered or re-issued tool call dedupes server-side via the gateway's idempotency path. The gateway only dedupes POST/PATCH, so only set it there. Both IDs are stable for a given tool call, making the key reproducible across re-attempts of the same call.
		var idempotencyKey string
		if (desc.Method == http.MethodPost || desc.Method == http.MethodPatch) && runCtx.RunID != "" && runCtx.ToolUseID != "" {
			idempotencyKey = runCtx.RunID + ":" + runCtx.ToolUseID
		}

		return runCtx.GatewayClient.Do(ctx, domain.GatewayRequest{
			Method:         desc.Method,
			Path:           path,
			Query:          query,
			Body:           bodyJSON,
			Identity:       runCtx.Identity,
			IdempotencyKey: idempotencyKey,
		})
	}
}

// jsonScalarToString renders a JSON scalar (string/number/bool) as a plain string for use in a path or query value, without surrounding quotes.
func jsonScalarToString(raw json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "null" {
		return "", nil
	}
	// Numbers and booleans render as their literal JSON form.
	return trimmed, nil
}

// appendQueryValue adds a query parameter. Array-typed query params are flattened to repeated key entries so the gateway's query binding sees each element.
func appendQueryValue(q url.Values, name string, raw json.RawMessage, isArray bool) error {
	if isArray {
		var elems []json.RawMessage
		if err := json.Unmarshal(raw, &elems); err != nil {
			return err
		}
		for _, e := range elems {
			s, err := jsonScalarToString(e)
			if err != nil {
				return err
			}
			q.Add(name, s)
		}
		return nil
	}
	s, err := jsonScalarToString(raw)
	if err != nil {
		return err
	}
	q.Add(name, s)
	return nil
}
