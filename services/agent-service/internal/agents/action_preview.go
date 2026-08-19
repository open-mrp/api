package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"github.com/augno/api/services/agent-service/internal/domain"
	"github.com/augno/api/shared/appnav"
)

// ActionPreview is a reviewable description of what a blocked tool call will do, built when the run
// pauses and carried on the pause step so both the chat approval card and the run console can render
// it without re-deriving anything.
//
// It exists because the raw tool input is not reviewable: `{"id":"acpr_x","unit_price":"11.00"}` says
// nothing about which price is being changed, what it is today, or that the other three fields are
// staying put. Approving an agent's write is a decision, and a decision needs the before as much as
// the after — so this resolves the target record and reads its current state, and every field the
// call sets is presented as current → new.
type ActionPreview struct {
	// Operation is the kind of change, for the reviewer's mental model: create, update, delete, action (a named operation like `issue` or `close`) or read.
	Operation string `json:"operation"`
	// Title names the operation the way the API does (the endpoint's display name, e.g. "Update Customer Price").
	Title string `json:"title"`
	// Resource identifies what is being changed. Absent on a create, which has no target yet.
	Resource *PreviewResource `json:"resource,omitempty"`
	// Fields are the values the call sets, in schema order, each with its current value where one could be read.
	Fields []PreviewField `json:"fields"`
	// Identifiers are the path parameters that select the target rather than change it, so the reviewer sees what was addressed without them cluttering the change list.
	Identifiers []PreviewField `json:"identifiers,omitempty"`
	Method      string         `json:"method"`
	Path        string         `json:"path"`
	// BeforeState reports whether current values were readable. False means the rows carry only new values — the reviewer must not read "no current value" as "this field is empty today".
	BeforeState bool `json:"before_state"`
	// Truncated is set when the call sets more fields than the preview carries.
	Truncated bool `json:"truncated,omitempty"`
}

// PreviewResource identifies the record an action targets, in the same shape the frontend's
// resource-link registry consumes — so the preview header can link straight to the record.
type PreviewResource struct {
	Object string `json:"object,omitempty"`
	ID     string `json:"id,omitempty"`
	// Label is the record's human name or number, read from the record itself.
	Label string `json:"label,omitempty"`
	// Linkable reports whether the frontend has a detail page for this object type.
	Linkable bool `json:"linkable"`
}

// PreviewField is one value a call sets, paired with the value it replaces.
type PreviewField struct {
	// Key is the dotted input path (`freight.carrier_id`), kept so a reviewer can tie a row back to the API field.
	Key string `json:"key"`
	// Label is the field's human name, including its parent group ("Freight › Carrier").
	Label string `json:"label"`
	// After is the value the call sets, and Before the value it replaces (absent when unread or unset).
	After  any `json:"after"`
	Before any `json:"before,omitempty"`
	// Changed is false when the call sets a field to the value it already holds — worth showing, but not as a change.
	Changed bool `json:"changed"`
	// Format hints at rendering (`decimal`, `date`, `date-time`, `id`, `enum`, `bool`).
	Format string `json:"format,omitempty"`
	// Description is the field's API documentation, for a reviewer who needs to know what the field means.
	Description string `json:"description,omitempty"`
}

// previewFieldLimit caps the rows one preview carries. A bulk call can set hundreds of values; past a
// point the card stops being reviewable anyway, and the reviewer is better served by an honest
// "and N more" than by a wall of rows.
const previewFieldLimit = 60

// BuildActionPreview describes a tool call the run is about to pause on.
//
// Never fails: a preview is a review aid, so anything unresolvable (an unparseable schema, an
// unreadable target) degrades to a thinner preview rather than blocking the pause. Returns nil for a
// tool with no catalog descriptor (a built-in tool, which has no endpoint to describe) or input that
// isn't a JSON object — the approval UI falls back to showing the raw input there.
func BuildActionPreview(ctx context.Context, slug string, rawInput json.RawMessage, runCtx *domain.HandlerRunContext) *ActionPreview {
	desc, ok := LookupEndpointTool(slug)
	if !ok {
		return nil
	}
	return buildActionPreview(ctx, desc, rawInput, runCtx)
}

func buildActionPreview(ctx context.Context, desc EndpointToolDescriptor, rawInput json.RawMessage, runCtx *domain.HandlerRunContext) *ActionPreview {
	input := map[string]any{}
	if len(rawInput) > 0 {
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return nil
		}
	}

	path, identifiers, body := splitInput(desc, input)
	schema := parseInputSchema(desc.InputSchema)

	p := &ActionPreview{
		Operation: classifyOperation(desc),
		Title:     desc.DisplayName,
		Method:    desc.Method,
		Path:      path,
	}

	// Current state, for the before column and the record's label. Only for operations that target an
	// existing record — a create has nothing to read, and the read costs a gateway round trip.
	var before map[string]any
	if p.Operation != "create" {
		before = fetchBeforeState(ctx, desc, path, runCtx)
		p.BeforeState = before != nil
	}

	p.Resource = describeTarget(identifiers, before)
	p.Identifiers = identifierFields(identifiers, schema)
	p.Fields, p.Truncated = previewFields(body, schema, before)

	return p
}

// splitInput separates the call's input the way the outgoing request does: path parameters address the
// target, and everything else is the payload. Returns the concrete path with parameters substituted.
func splitInput(desc EndpointToolDescriptor, input map[string]any) (path string, identifiers map[string]any, body map[string]any) {
	path = desc.RouteTemplate
	identifiers = map[string]any{}
	body = map[string]any{}

	byName := make(map[string]EndpointToolParam, len(desc.Params))
	for _, prm := range desc.Params {
		byName[prm.Name] = prm
	}

	for name, val := range input {
		switch byName[name].In {
		case EndpointToolParamPath:
			identifiers[name] = val
			path = strings.ReplaceAll(path, "{"+name+"}", scalarString(val))
		case EndpointToolParamQuery:
			// `include` only shapes the response; it changes nothing and would read as a field being set.
			if name == "include" {
				continue
			}
			body[name] = val
		default:
			body[name] = val
		}
	}
	return path, identifiers, body
}

// classifyOperation names what the call does, from the endpoint's method and route shape. The route is
// the reliable signal: this API expresses named operations as `/actions/<verb>` sub-routes and creates
// as a POST to a collection, so "PUT means update" alone would mislabel both.
func classifyOperation(desc EndpointToolDescriptor) string {
	if !desc.Mutating() {
		return "read"
	}
	route := desc.RouteTemplate
	if desc.Method == "DELETE" {
		return "delete"
	}
	if strings.Contains(route, "/actions/") {
		return "action"
	}
	if desc.Method == "POST" && !strings.HasSuffix(route, "}") {
		return "create"
	}
	return "update"
}

// fetchBeforeState reads the target record so the preview can show current values.
//
// The read goes through the same gateway client and agent identity as the call itself, so it cannot
// see more than the agent can; a denied or missing read simply yields no before column. Only a GET is
// ever issued, so this cannot itself change anything.
func fetchBeforeState(ctx context.Context, desc EndpointToolDescriptor, path string, runCtx *domain.HandlerRunContext) map[string]any {
	if runCtx == nil || runCtx.GatewayClient == nil {
		return nil
	}
	detail := detailPath(path)
	if detail == "" {
		return nil
	}

	raw, err := runCtx.GatewayClient.Do(ctx, domain.GatewayRequest{
		Method:   "GET",
		Path:     detail,
		Identity: runCtx.Identity,
	})
	if err != nil {
		// Expected whenever the resource has no readable detail route or the agent lacks the read scope.
		slog.Debug("Action preview: current state unavailable", "tool", desc.Slug, "path", detail, "error", err)
		return nil
	}
	var state map[string]any
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return nil
	}
	return state
}

// detailPath turns an operation's route into the route that reads the record it operates on: an
// `/actions/<verb>` suffix is dropped, and anything after the addressed id is a sub-resource the
// action reaches through. Empty when the route addresses no single record (a collection POST).
func detailPath(path string) string {
	if i := strings.Index(path, "/actions/"); i >= 0 {
		path = path[:i]
	}
	path = strings.TrimSuffix(path, "/")
	// A route still holding an unsubstituted parameter was addressed with an input we don't have; reading it would 404.
	if strings.Contains(path, "{") {
		return ""
	}
	return path
}

// describeTarget identifies the record being changed, preferring what the record says about itself
// over what the call's input implies.
func describeTarget(identifiers map[string]any, before map[string]any) *PreviewResource {
	res := &PreviewResource{}
	if before != nil {
		res.Object = stringField(before, "object")
		res.ID = stringField(before, "id")
		res.Label = recordLabel(before)
	}
	if res.ID == "" {
		// Fall back to the id the call addressed. `id` is the conventional name for the primary target;
		// a route with a single path parameter under another name (`order_id`) is that target too.
		if v, ok := identifiers["id"]; ok {
			res.ID = scalarString(v)
		} else if len(identifiers) == 1 {
			for _, v := range identifiers {
				res.ID = scalarString(v)
			}
		}
	}
	if res.Object == "" && res.ID == "" && res.Label == "" {
		return nil
	}
	if res.Object != "" {
		_, res.Linkable = appnav.RecordRouteFor(res.Object)
	}
	return res
}

// labelFields are the fields records in this API use to name themselves, most human-recognizable first.
var labelFields = []string{"number", "name", "display_name", "title", "code", "email", "slug", "reference"}

func recordLabel(state map[string]any) string {
	for _, f := range labelFields {
		if v := stringField(state, f); v != "" {
			return v
		}
	}
	return ""
}

// identifierFields renders the path parameters as rows, so a reviewer can see exactly what was addressed.
func identifierFields(identifiers map[string]any, schema *propSchema) []PreviewField {
	names := slices.Sorted(maps.Keys(identifiers))
	out := make([]PreviewField, 0, len(names))
	for _, name := range names {
		out = append(out, PreviewField{
			Key:         name,
			Label:       humanizeKey(name),
			After:       identifiers[name],
			Format:      "id",
			Description: schema.descriptionOf(name),
		})
	}
	return out
}

// previewFields flattens the payload into reviewable rows, pairing each with its current value.
//
// Order follows the input rather than the schema: an agent sends the fields it means to change, and
// keeping that order keeps the rows in the order the change was expressed.
func previewFields(body map[string]any, schema *propSchema, before map[string]any) ([]PreviewField, bool) {
	var fields []PreviewField
	truncated := false

	for _, name := range slices.Sorted(maps.Keys(body)) {
		if len(fields) >= previewFieldLimit {
			truncated = true
			break
		}
		flatten(name, humanizeKey(name), body[name], schema.child(name), pluck(before, name), before != nil, &fields, &truncated)
	}
	return fields, truncated
}

// flatten expands one input value into rows: an object contributes a row per leaf under a grouped
// label, an array of objects a row per element field, and anything else a single row. Nesting is
// flattened rather than kept as a tree because a reviewer reads a change as a list of fields, and a
// dotted path plus a "Parent › Child" label carries the structure without needing one.
func flatten(key, label string, value any, schema *propSchema, before any, haveBefore bool, out *[]PreviewField, truncated *bool) {
	if len(*out) >= previewFieldLimit {
		*truncated = true
		return
	}

	switch v := value.(type) {
	case map[string]any:
		beforeMap, _ := before.(map[string]any)
		for _, name := range slices.Sorted(maps.Keys(v)) {
			var childBefore any
			if beforeMap != nil {
				childBefore = beforeMap[name]
			}
			flatten(key+"."+name, label+" › "+humanizeKey(name), v[name], schema.child(name), childBefore, haveBefore, out, truncated)
		}
		return
	case []any:
		// An array of scalars reads better as one row than as a row per element.
		if !containsObject(v) {
			appendField(out, truncated, PreviewField{
				Key: key, Label: label,
				After:       joinScalars(v),
				Before:      beforeScalar(before, haveBefore),
				Changed:     changed(joinScalars(v), beforeScalar(before, haveBefore), haveBefore),
				Description: schema.description,
			})
			return
		}
		beforeList, _ := before.([]any)
		for i, elem := range v {
			var elemBefore any
			if i < len(beforeList) {
				elemBefore = beforeList[i]
			}
			flatten(fmt.Sprintf("%s.%d", key, i), fmt.Sprintf("%s %d", singularLabel(label), i+1), elem, schema.item(), elemBefore, haveBefore, out, truncated)
		}
		return
	}

	beforeVal := beforeScalar(before, haveBefore)
	appendField(out, truncated, PreviewField{
		Key: key, Label: label,
		After:       value,
		Before:      beforeVal,
		Changed:     changed(value, beforeVal, haveBefore),
		Format:      schema.format(),
		Description: schema.description,
	})
}

func appendField(out *[]PreviewField, truncated *bool, f PreviewField) {
	if len(*out) >= previewFieldLimit {
		*truncated = true
		return
	}
	*out = append(*out, f)
}

// changed reports whether a field's new value differs from its current one. Without a before state
// nothing can be called unchanged, so every field is a change — the safe reading when we don't know.
func changed(after, before any, haveBefore bool) bool {
	if !haveBefore {
		return true
	}
	return !sameValue(after, before)
}

// sameValue compares two JSON values by their canonical encoding, so a decimal the API returns as
// "11.00" and one the agent sent as "11.00" match while 11 and "11.00" do not claim to.
func sameValue(a, b any) bool {
	ab, errA := json.Marshal(a)
	bb, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	return string(ab) == string(bb)
}

func beforeScalar(before any, haveBefore bool) any {
	if !haveBefore {
		return nil
	}
	return before
}

func containsObject(list []any) bool {
	return slices.ContainsFunc(list, func(e any) bool {
		_, ok := e.(map[string]any)
		return ok
	})
}

func joinScalars(list []any) string {
	parts := make([]string, 0, len(list))
	for _, e := range list {
		parts = append(parts, scalarString(e))
	}
	return strings.Join(parts, ", ")
}

// pluck reads a top-level field from the current state, tolerating a nil state.
func pluck(state map[string]any, key string) any {
	if state == nil {
		return nil
	}
	return state[key]
}

func stringField(state map[string]any, key string) string {
	s, _ := state[key].(string)
	return s
}

func scalarString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case json.RawMessage:
		return strings.Trim(string(t), `"`)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return strings.Trim(string(b), `"`)
	}
}

// smallWords stay lowercase inside a label; anything else is capitalized.
var smallWords = map[string]bool{"of": true, "to": true, "for": true, "at": true, "in": true, "on": true, "by": true, "per": true}

// humanizeKey turns an API field name into a label: `unit_price` → "Unit price", `id` → "ID". Sentence
// case, not title case, because the API's own field documentation reads that way.
func humanizeKey(key string) string {
	words := strings.Split(strings.ReplaceAll(key, "-", "_"), "_")
	for i, w := range words {
		switch {
		case w == "":
			continue
		case w == "id":
			words[i] = "ID"
		case w == "url" || w == "sku" || w == "po" || w == "ein" || w == "ap" || w == "ar":
			words[i] = strings.ToUpper(w)
		case i == 0:
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		case smallWords[w]:
			// leave lowercase
		default:
			words[i] = w
		}
	}
	return strings.Join(words, " ")
}

// singularLabel turns a collection label into an element label ("Lines" → "Line") for numbering array rows.
func singularLabel(label string) string {
	if strings.HasSuffix(label, "ies") {
		return strings.TrimSuffix(label, "ies") + "y"
	}
	if base, ok := strings.CutSuffix(label, "s"); ok && !strings.HasSuffix(base, "s") {
		return base
	}
	return label
}

// ---------------------------------------------------------------------------------------------
// Input schema

// propSchema is the slice of JSON Schema the preview needs: what a field means, and how to render it.
type propSchema struct {
	typ         string
	fmt         string
	description string
	enum        []string
	properties  map[string]*propSchema
	items       *propSchema
}

func parseInputSchema(raw string) *propSchema {
	if raw == "" {
		return &propSchema{}
	}
	var node schemaNode
	if err := json.Unmarshal([]byte(raw), &node); err != nil {
		return &propSchema{}
	}
	return node.toProp()
}

type schemaNode struct {
	Type        string                 `json:"type"`
	Format      string                 `json:"format"`
	Description string                 `json:"description"`
	Enum        []string               `json:"enum"`
	Properties  map[string]*schemaNode `json:"properties"`
	Items       *schemaNode            `json:"items"`
}

func (n *schemaNode) toProp() *propSchema {
	if n == nil {
		return &propSchema{}
	}
	p := &propSchema{typ: n.Type, fmt: n.Format, description: n.Description, enum: n.Enum}
	if len(n.Properties) > 0 {
		p.properties = make(map[string]*propSchema, len(n.Properties))
		for k, v := range n.Properties {
			p.properties[k] = v.toProp()
		}
	}
	if n.Items != nil {
		p.items = n.Items.toProp()
	}
	return p
}

func (p *propSchema) child(name string) *propSchema {
	if p == nil || p.properties == nil {
		return &propSchema{}
	}
	if c, ok := p.properties[name]; ok {
		return c
	}
	return &propSchema{}
}

func (p *propSchema) item() *propSchema {
	if p == nil || p.items == nil {
		return &propSchema{}
	}
	return p.items
}

func (p *propSchema) descriptionOf(name string) string {
	return p.child(name).description
}

// format is the rendering hint for a value: the schema's own `format` when it has one, else what the
// declared type implies.
func (p *propSchema) format() string {
	if p == nil {
		return ""
	}
	if p.fmt != "" {
		return p.fmt
	}
	if len(p.enum) > 0 {
		return "enum"
	}
	switch p.typ {
	case "boolean":
		return "bool"
	case "integer", "number":
		return "number"
	default:
		return ""
	}
}
