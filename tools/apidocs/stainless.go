package main

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"runtime/debug"
	"sort"
	"strings"
	"unicode"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	"gopkg.in/yaml.v3"
)

type stainlessNode struct {
	methods          map[string]string
	methodOrder      []string
	subresources     map[string]*stainlessNode
	subresourceOrder []string
	models           map[string]string
}

type prefixStats struct {
	exactMethods             map[string]bool
	descendantRoutes         bool
	descendantStaticAfter    bool
	childParamMethods        map[string]bool
	childParamHasDescendants bool
}

type endpointSDKMeta struct {
	group           apiendpoint.APIEndpointGroup
	specField       reflect.Value
	method          string
	route           string
	segments        []string
	nodePrefix      []string
	resourcePath    []string
	methodKey       string
	modelSchemaRefs []string
}

func newStainlessNode() *stainlessNode {
	return &stainlessNode{
		methods:      make(map[string]string),
		subresources: make(map[string]*stainlessNode),
		models:       make(map[string]string),
	}
}

func (n *stainlessNode) child(name string) *stainlessNode {
	if child, ok := n.subresources[name]; ok {
		return child
	}

	child := newStainlessNode()
	n.subresources[name] = child
	n.subresourceOrder = append(n.subresourceOrder, name)
	return child
}

func (n *stainlessNode) setMethod(key, endpoint string) error {
	if existing, ok := n.methods[key]; ok {
		if existing == endpoint {
			return nil
		}
		return fmt.Errorf("duplicate stainless method key %q for %q and %q", key, existing, endpoint)
	}

	n.methods[key] = endpoint
	n.methodOrder = append(n.methodOrder, key)
	return nil
}

func (n *stainlessNode) addModels(refs []string) {
	for _, ref := range refs {
		if ref == "" {
			continue
		}
		n.models[schemaNameToAlias(ref)] = "#/components/schemas/" + ref
	}
}

// dedupeModels enforces that each model alias is declared exactly once across
// the whole resource tree. Endpoints collect their full transitive schema
// closure, so shared schemas (Account, Owner, PageInfo, ...) would otherwise be
// declared under every resource that references them. Stainless treats model
// names as global, and duplicate declarations fail the build with
// Model/DuplicateName. We claim each alias at the first node that references it
// — walking parents before children in declaration order — and drop the
// declaration from every later node.
func dedupeModels(root *stainlessNode) {
	claimed := make(map[string]bool)

	var walk func(n *stainlessNode)
	walk = func(n *stainlessNode) {
		for alias := range n.models {
			if claimed[alias] {
				delete(n.models, alias)
				continue
			}
			claimed[alias] = true
		}
		for _, name := range n.subresourceOrder {
			walk(n.subresources[name])
		}
	}

	walk(root)
}

func generateStainlessConfigs(groups []apiendpoint.APIEndpointGroup, version string) error {
	workspaces := []struct {
		outputPath string
		publicOnly bool
	}{
		{outputPath: "stainless/public/stainless.yml", publicOnly: true},
		{outputPath: "stainless/internal/stainless.yml", publicOnly: false},
	}

	for _, workspace := range workspaces {
		if err := generateStainlessConfig(groups, workspace.outputPath, workspace.publicOnly, version); err != nil {
			return fmt.Errorf("generating Stainless config %s: %w", workspace.outputPath, err)
		}
	}
	return nil
}

func generateStainlessConfig(groups []apiendpoint.APIEndpointGroup, outputPath string, publicOnly bool, version string) (err error) {
	// Schema generation reports invariant violations by panicking deep inside
	// the recursive builder; surface those as returned errors instead of
	// crashing the process (main() owns the exit).
	defer func() {
		if r := recover(); r != nil {
			// No path prefix here: the caller wraps with the path-qualified
			// "generating Stainless config %s" context.
			err = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
		}
	}()

	spec, _, err := buildOpenAPISpec(groups, publicOnly, version)
	if err != nil {
		return err
	}

	metas, err := collectEndpointSDKMetadata(groups, publicOnly, spec.Components.Schemas)
	if err != nil {
		return err
	}

	root := newStainlessNode()
	for _, meta := range metas {
		node := root
		for _, segment := range meta.resourcePath {
			node = node.child(segment)
		}
		if err := node.setMethod(meta.methodKey, strings.ToLower(meta.method)+" "+meta.route); err != nil {
			return err
		}
		node.addModels(meta.modelSchemaRefs)
	}

	dedupeModels(root)

	if err := rewriteStainlessResources(outputPath, root); err != nil {
		return err
	}

	// Keep the OpenMRP-Version default header in sync with the spec's info.version
	// (which is set to `version` in buildOpenAPISpec). The self-hosted stlc fork
	// emits this header literally, so it must be refreshed on every generation.
	if err := syncVersionHeaderValue(outputPath, version); err != nil {
		return err
	}

	specType := "internal"
	if publicOnly {
		specType = "public"
	}
	logInfof("Stainless config generated in %s (%d %s endpoints)\n", outputPath, len(metas), specType)
	return nil
}

func collectEndpointSDKMetadata(groups []apiendpoint.APIEndpointGroup, publicOnly bool, schemas map[string]Schema) ([]endpointSDKMeta, error) {
	rawMetas := make([]endpointSDKMeta, 0)
	prefixMap := make(map[string]*prefixStats)
	nodePrefixes := make(map[string][]string)

	for _, group := range groups {
		for _, endpoint := range group.Endpoints {
			specField := endpointSpecField(endpoint)
			if publicOnly && !specField.FieldByName("Public").Bool() {
				continue
			}

			route := strings.TrimSpace(specField.FieldByName("Route").String())
			method := strings.ToUpper(strings.TrimSpace(specField.FieldByName("Method").String()))
			if route == "" || method == "" {
				return nil, fmt.Errorf("endpoint %T has empty route or method", endpoint)
			}

			segments := splitRouteSegments(route)
			if len(segments) == 0 {
				return nil, fmt.Errorf("endpoint %T has empty route segments for %q", endpoint, route)
			}

			meta := endpointSDKMeta{
				group:           group,
				specField:       specField,
				method:          method,
				route:           route,
				segments:        segments,
				modelSchemaRefs: collectEndpointSchemaRefs(endpoint, method, schemas),
			}
			rawMetas = append(rawMetas, meta)

			recordPromotablePrefixes(prefixMap, nodePrefixes, segments, method)
		}
	}

	for i := range rawMetas {
		meta := &rawMetas[i]
		meta.nodePrefix = findResourceNodePrefix(meta.segments, prefixMap, nodePrefixes)
		meta.resourcePath = resourcePathForMeta(*meta)
		meta.methodKey = methodKeyForMeta(*meta)
		if meta.methodKey == "" {
			return nil, fmt.Errorf("could not derive stainless method key for %s %s", meta.method, meta.route)
		}
	}

	return rawMetas, nil
}

func recordPromotablePrefixes(prefixMap map[string]*prefixStats, nodePrefixes map[string][]string, segments []string, method string) {
	for i, segment := range segments {
		if isRouteParam(segment) {
			continue
		}

		prefix := append([]string(nil), segments[:i+1]...)
		key := routePrefixKey(prefix)
		if _, ok := nodePrefixes[key]; !ok {
			nodePrefixes[key] = prefix
		}

		stats, ok := prefixMap[key]
		if !ok {
			stats = &prefixStats{
				exactMethods:      make(map[string]bool),
				childParamMethods: make(map[string]bool),
			}
			prefixMap[key] = stats
		}

		remaining := segments[i+1:]
		switch {
		case len(remaining) == 0:
			stats.exactMethods[method] = true
		default:
			stats.descendantRoutes = true
			if isRouteParam(remaining[0]) {
				stats.childParamMethods[method] = true
				if len(remaining) > 1 {
					stats.childParamHasDescendants = true
				}
			} else {
				stats.descendantStaticAfter = true
			}
		}
	}
}

func findResourceNodePrefix(segments []string, prefixMap map[string]*prefixStats, nodePrefixes map[string][]string) []string {
	best := []string{segments[0]}
	bestLen := 1

	for key, stats := range prefixMap {
		prefix := nodePrefixes[key]
		if !hasPrefixSegments(segments, prefix) {
			continue
		}
		if !shouldPromotePrefix(prefix, stats) {
			continue
		}
		if len(prefix) > bestLen {
			best = prefix
			bestLen = len(prefix)
		}
	}

	return append([]string(nil), best...)
}

func shouldPromotePrefix(prefix []string, stats *prefixStats) bool {
	if len(prefix) == 1 {
		return true
	}
	if prefix[len(prefix)-1] == "actions" {
		return true
	}
	if len(stats.exactMethods) > 1 {
		return true
	}
	if len(stats.exactMethods) > 0 && stats.descendantRoutes {
		return true
	}
	if stats.descendantStaticAfter {
		return true
	}
	if len(stats.childParamMethods) > 1 {
		return true
	}
	if stats.childParamHasDescendants {
		return true
	}
	return false
}

func resourcePathForMeta(meta endpointSDKMeta) []string {
	if len(meta.group.SDKResourcePath) > 0 {
		out := make([]string, 0, len(meta.group.SDKResourcePath))
		for _, segment := range meta.group.SDKResourcePath {
			out = append(out, normalizeIdentifier(segment))
		}
		return out
	}

	out := make([]string, 0, len(meta.nodePrefix))
	for _, segment := range meta.nodePrefix {
		if isRouteParam(segment) {
			continue
		}
		out = append(out, normalizeIdentifier(segment))
	}
	return out
}

func methodKeyForMeta(meta endpointSDKMeta) string {
	override := strings.TrimSpace(meta.specField.FieldByName("SDKMethodKey").String())
	if override != "" {
		return normalizeIdentifier(override)
	}

	if len(meta.nodePrefix) > 0 && meta.nodePrefix[len(meta.nodePrefix)-1] == "actions" {
		leaf := normalizeIdentifier(lastStaticSegment(meta.segments[len(meta.nodePrefix):]))
		if leaf != "" {
			return leaf
		}
	}

	remaining := meta.segments[len(meta.nodePrefix):]
	if len(remaining) == 0 {
		return canonicalExactMethodKey(meta.method)
	}

	if len(remaining) == 1 && isRouteParam(remaining[0]) {
		return canonicalItemMethodKey(meta.method)
	}

	leaf := normalizeIdentifier(lastStaticSegment(remaining))
	if leaf == "" {
		return canonicalExactMethodKey(meta.method)
	}

	if meta.method == "POST" {
		return leaf
	}

	prefix := actionVerbForMethod(meta.method)
	if prefix == "" {
		return leaf
	}
	return prefix + "_" + leaf
}

func canonicalExactMethodKey(method string) string {
	switch method {
	case "GET":
		return "list"
	case "POST":
		return "create"
	case "PATCH", "PUT":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return normalizeIdentifier(strings.ToLower(method))
	}
}

func canonicalItemMethodKey(method string) string {
	switch method {
	case "GET":
		return "retrieve"
	case "PATCH", "PUT":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return canonicalExactMethodKey(method)
	}
}

func actionVerbForMethod(method string) string {
	switch method {
	case "GET":
		return "retrieve"
	case "PATCH", "PUT":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return ""
	}
}

func collectEndpointSchemaRefs(endpoint apiendpoint.APIEndpointer, method string, schemas map[string]Schema) []string {
	refs := make(map[string]bool)
	addRootSchemaRef(refs, endpoint.GetResponseType(), schemas)

	if method != "GET" && method != "DELETE" && endpointRequestHasJSONFields(endpoint.GetRequestType()) {
		addRootSchemaRef(refs, endpoint.GetRequestType(), schemas)
	}

	out := make([]string, 0, len(refs))
	for ref := range refs {
		out = append(out, ref)
	}
	sort.Strings(out)
	return out
}

func addRootSchemaRef(refs map[string]bool, rootType reflect.Type, schemas map[string]Schema) {
	schemaName := getCleanTypeName(rootType)
	if schemaName == "" || schemaName == "EmptyResource" {
		return
	}

	visited := make(map[string]bool)
	collectSchemaRefs(schemaName, schemas, refs, visited)
}

func collectSchemaRefs(name string, schemas map[string]Schema, refs map[string]bool, visited map[string]bool) {
	if name == "" || visited[name] {
		return
	}
	visited[name] = true

	schema, ok := schemas[name]
	if !ok {
		refs[name] = true
		return
	}

	refs[name] = true
	collectSchemaRefsFromSchema(schema, schemas, refs, visited)
}

func collectSchemaRefsFromSchema(schema Schema, schemas map[string]Schema, refs map[string]bool, visited map[string]bool) {
	if refName := schemaRefName(schema.Ref); refName != "" {
		collectSchemaRefs(refName, schemas, refs, visited)
	}

	if schema.Items != nil {
		collectSchemaRefsFromSchema(*schema.Items, schemas, refs, visited)
	}
	if schema.AdditionalProperties != nil {
		collectSchemaRefsFromSchema(*schema.AdditionalProperties, schemas, refs, visited)
	}
	for _, property := range schema.Properties {
		collectSchemaRefsFromSchema(property, schemas, refs, visited)
	}
	for _, option := range schema.OneOf {
		collectSchemaRefsFromSchema(option, schemas, refs, visited)
	}
	for _, option := range schema.AnyOf {
		collectSchemaRefsFromSchema(option, schemas, refs, visited)
	}
	for _, option := range schema.AllOf {
		collectSchemaRefsFromSchema(option, schemas, refs, visited)
	}
}

func rewriteStainlessResources(path string, resources *stainlessNode) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return err
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("unexpected yaml document shape in %s", path)
	}

	root := doc.Content[0]
	resourcesNode := buildResourcesYAMLNode(resources)
	replaceTopLevelMappingNode(root, "resources", resourcesNode)

	var formatted bytes.Buffer
	encoder := yaml.NewEncoder(&formatted)
	encoder.SetIndent(2)
	if err := encoder.Encode(&doc); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}

	return os.WriteFile(path, formatted.Bytes(), 0600)
}

func replaceTopLevelMappingNode(root *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			root.Content[i+1] = value
			return
		}
	}

	root.Content = append(root.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		value,
	)
}

// syncVersionHeaderValue rewrites the value of every
// client_settings.default_headers entry marked `version_header: true` to the
// given API version. The header's name and presence stay hand-maintained in
// stainless.yml; this only keeps its value aligned with the spec's info.version
// so the generated SDKs send the correct OpenMRP-Version header. It is a no-op if
// no such header is configured.
func syncVersionHeaderValue(path, version string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return err
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("unexpected yaml document shape in %s", path)
	}

	defaultHeaders := mappingValue(mappingValue(doc.Content[0], "client_settings"), "default_headers")
	if defaultHeaders == nil || defaultHeaders.Kind != yaml.MappingNode {
		return nil
	}

	changed := false
	for i := 0; i+1 < len(defaultHeaders.Content); i += 2 {
		header := defaultHeaders.Content[i+1]
		if header.Kind != yaml.MappingNode {
			continue
		}
		if marker := mappingValue(header, "version_header"); marker == nil || marker.Value != "true" {
			continue
		}
		setMappingStringValue(header, "value", version)
		changed = true
	}
	if !changed {
		return nil
	}

	var formatted bytes.Buffer
	encoder := yaml.NewEncoder(&formatted)
	encoder.SetIndent(2)
	if err := encoder.Encode(&doc); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}

	return os.WriteFile(path, formatted.Bytes(), 0600)
}

// mappingValue returns the value node associated with key in a mapping node, or
// nil if node is not a mapping or the key is absent.
func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// setMappingStringValue sets (or inserts) a double-quoted string scalar for key
// in a mapping node.
func setMappingStringValue(node *yaml.Node, key, value string) {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			node.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle, Value: value}
			return
		}
	}
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle, Value: value},
	)
}

func buildResourcesYAMLNode(root *stainlessNode) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, key := range root.subresourceOrder {
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
			buildStainlessNodeYAML(root.subresources[key]),
		)
	}
	return node
}

func buildStainlessNodeYAML(node *stainlessNode) *yaml.Node {
	out := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	if len(node.methods) > 0 {
		methodsNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for _, key := range node.methodOrder {
			methodsNode.Content = append(methodsNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: node.methods[key]},
			)
		}
		out.Content = append(out.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "methods"},
			methodsNode,
		)
	}

	if len(node.subresources) > 0 {
		subresourcesNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for _, key := range node.subresourceOrder {
			subresourcesNode.Content = append(subresourcesNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
				buildStainlessNodeYAML(node.subresources[key]),
			)
		}
		out.Content = append(out.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "subresources"},
			subresourcesNode,
		)
	}

	if len(node.models) > 0 {
		modelKeys := make([]string, 0, len(node.models))
		for key := range node.models {
			modelKeys = append(modelKeys, key)
		}
		sort.Strings(modelKeys)

		modelsNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for _, key := range modelKeys {
			modelsNode.Content = append(modelsNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: node.models[key]},
			)
		}
		out.Content = append(out.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "models"},
			modelsNode,
		)
	}

	return out
}

func splitRouteSegments(route string) []string {
	trimmed := strings.Trim(strings.TrimSpace(route), "/")
	if trimmed == "" {
		return nil
	}

	segments := strings.Split(trimmed, "/")
	if len(segments) > 0 && segments[0] == "v1" {
		segments = segments[1:]
	}
	return segments
}

func hasPrefixSegments(segments, prefix []string) bool {
	if len(prefix) > len(segments) {
		return false
	}
	for i := range prefix {
		if segments[i] != prefix[i] {
			return false
		}
	}
	return true
}

func routePrefixKey(segments []string) string {
	return strings.Join(segments, "/")
}

func isRouteParam(segment string) bool {
	return strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}")
}

func lastStaticSegment(segments []string) string {
	for i := len(segments) - 1; i >= 0; i-- {
		if !isRouteParam(segments[i]) {
			return segments[i]
		}
	}
	return ""
}

func schemaRefName(ref string) string {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return ""
	}
	return strings.TrimPrefix(ref, prefix)
}

func schemaNameToAlias(name string) string {
	return normalizeIdentifier(name)
}

func normalizeIdentifier(input string) string {
	if input == "" {
		return ""
	}

	runes := []rune(input)
	var out []rune
	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			if len(out) > 0 && out[len(out)-1] != '_' {
				out = append(out, '_')
			}
			continue
		}

		if len(out) > 0 && out[len(out)-1] != '_' && shouldInsertIdentifierUnderscore(runes, i) {
			out = append(out, '_')
		}

		out = append(out, unicode.ToLower(r))
	}

	return strings.Trim(strings.ReplaceAll(string(out), "__", "_"), "_")
}

func shouldInsertIdentifierUnderscore(runes []rune, idx int) bool {
	curr := runes[idx]
	prev, ok := previousIdentifierRune(runes, idx)
	if !ok {
		return false
	}

	next, hasNext := nextIdentifierRune(runes, idx)

	switch {
	case unicode.IsUpper(curr) && (unicode.IsLower(prev) || unicode.IsDigit(prev)):
		return true
	case unicode.IsUpper(curr) && unicode.IsUpper(prev) && hasNext && unicode.IsLower(next):
		return consecutiveUpperCount(runes, idx-1) > 1
	case unicode.IsDigit(curr) && !unicode.IsDigit(prev):
		return true
	case !unicode.IsDigit(curr) && unicode.IsDigit(prev):
		return true
	default:
		return false
	}
}

func previousIdentifierRune(runes []rune, idx int) (rune, bool) {
	for i := idx - 1; i >= 0; i-- {
		if unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) {
			return runes[i], true
		}
	}
	return 0, false
}

func nextIdentifierRune(runes []rune, idx int) (rune, bool) {
	for i := idx + 1; i < len(runes); i++ {
		if unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) {
			return runes[i], true
		}
	}
	return 0, false
}

func consecutiveUpperCount(runes []rune, idx int) int {
	count := 0
	for i := idx; i >= 0; i-- {
		if !unicode.IsUpper(runes[i]) {
			break
		}
		count++
	}
	return count
}
