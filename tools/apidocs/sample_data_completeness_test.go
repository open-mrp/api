package main

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/augno/api/shared/version"
	"github.com/stretchr/testify/require"
)

func jsonNormalizeExample(ex any) any {
	if ex == nil {
		return nil
	}
	b, err := json.Marshal(ex)
	if err != nil {
		return ex
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return ex
	}
	return out
}

func TestSampleDataCompleteness(t *testing.T) {
	groups := openAPIEndpointGroups()

	for _, tc := range []struct {
		name       string
		publicOnly bool
	}{
		{"public", true},
		{"internal", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := buildOpenAPIDocument(groups, tc.publicOnly, nil, version.Latest.Version)
			require.NoError(t, err)

			paths, ok := doc["paths"].(map[string]any)
			require.True(t, ok)

			for route, methods := range paths {
				mm, ok := methods.(map[string]any)
				require.True(t, ok, "path %q methods", route)

				for method, opAny := range mm {
					op, ok := opAny.(map[string]any)
					require.True(t, ok, "%s %s", method, route)

					opID, _ := op["operationId"].(string)
					ctx := strings.ToUpper(method) + " " + route
					if opID != "" {
						ctx = opID + " (" + ctx + ")"
					}

					assertOperationParameters(t, ctx, op)
					assertOperationRequestBody(t, ctx, op)
					assertOperationSuccessResponses(t, ctx, op, route, method)
				}
			}
		})
	}
}

func assertOperationParameters(t *testing.T, ctx string, op map[string]any) {
	raw, ok := op["parameters"]
	if !ok {
		return
	}
	arr, ok := raw.([]any)
	if !ok {
		return
	}

	for _, pAny := range arr {
		p, ok := pAny.(map[string]any)
		if !ok {
			continue
		}
		in, _ := p["in"].(string)
		name, _ := p["name"].(string)

		switch in {
		case "path":
			if pathExampleEmpty(p["example"]) {
				t.Errorf("%s: path parameter %q has empty or missing example", ctx, name)
			}
		case "query":
			ex, has := p["example"]
			if !has {
				t.Errorf("%s: query parameter %q missing example", ctx, name)
				continue
			}
			if weakQueryExample(ex) {
				t.Errorf("%s: query parameter %q has weak example %#v", ctx, name, ex)
			}
		}
	}
}

func pathExampleEmpty(ex any) bool {
	if ex == nil {
		return true
	}
	s, ok := ex.(string)
	if ok {
		return strings.TrimSpace(s) == ""
	}
	return false
}

func weakQueryExample(ex any) bool {
	switch v := ex.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == "" || v == "example"
	case []any:
		return len(v) == 0
	case float64, bool:
		return false
	default:
		return false
	}
}

func assertOperationRequestBody(t *testing.T, ctx string, op map[string]any) {
	rb, ok := op["requestBody"].(map[string]any)
	if !ok {
		return
	}
	content, ok := rb["content"].(map[string]any)
	if !ok {
		return
	}
	media, ok := content["application/json"].(map[string]any)
	if !ok {
		return
	}

	rawEx, has := media["example"]
	if !has || rawEx == nil {
		t.Errorf("%s: request body missing application/json example", ctx)
		return
	}
	ex := jsonNormalizeExample(rawEx)

	allow := documentedAugnoIDs()
	walkAugnoLikeStrings(ex, func(id string) {
		if _, ok := allow[id]; !ok {
			t.Errorf("%s: request body example contains undocumented id-like string %q", ctx, id)
		}
	})
}

func assertOperationSuccessResponses(t *testing.T, ctx string, op map[string]any, route, method string) {
	responses, ok := op["responses"].(map[string]any)
	if !ok {
		return
	}

	for code, respAny := range responses {
		if len(code) == 0 || code[0] != '2' {
			continue
		}
		if _, err := strconv.Atoi(code); err != nil {
			continue
		}

		resp, ok := respAny.(map[string]any)
		if !ok {
			continue
		}
		content, ok := resp["content"].(map[string]any)
		if !ok {
			continue
		}
		media, ok := content["application/json"].(map[string]any)
		if !ok {
			continue
		}

		rawEx, has := media["example"]
		if !has || rawEx == nil {
			t.Errorf("%s: %s response missing application/json example", ctx, code)
			continue
		}
		ex := jsonNormalizeExample(rawEx)

		allow := documentedAugnoIDs()
		walkAugnoLikeStrings(ex, func(id string) {
			if _, ok := allow[id]; !ok {
				t.Errorf("%s: response example contains undocumented id-like string %q", ctx, id)
			}
		})

		assertPathResponseIDCoherence(t, ctx, op, route, method, ex)
	}
}

func assertPathResponseIDCoherence(t *testing.T, ctx string, op map[string]any, route, method string, respEx any) {
	if !resourceCoherenceApplies(route, method) {
		return
	}
	param, ok := singlePathTemplate(route)
	if !ok {
		return
	}
	respMap, ok := respEx.(map[string]any)
	if !ok {
		return
	}
	respID, ok := respMap["id"].(string)
	if !ok || respID == "" {
		return
	}

	pathVal, ok := lookupPathParameterExample(op, param)
	if !ok || pathVal == "" {
		return
	}
	if pathVal != respID {
		t.Errorf("%s: path parameter %q example %q must match response id %q", ctx, param, pathVal, respID)
	}
}

func resourceCoherenceApplies(route, method string) bool {
	param, ok := singlePathTemplate(route)
	if !ok || param != "id" {
		return false
	}
	switch strings.ToUpper(method) {
	case "GET", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func singlePathTemplate(route string) (string, bool) {
	if strings.Count(route, "{") != 1 || strings.Count(route, "}") != 1 {
		return "", false
	}
	i := strings.Index(route, "{")
	j := strings.Index(route, "}")
	if i < 0 || j < i {
		return "", false
	}
	return route[i+1 : j], true
}

func lookupPathParameterExample(op map[string]any, param string) (string, bool) {
	raw, ok := op["parameters"]
	if !ok {
		return "", false
	}
	arr, ok := raw.([]any)
	if !ok {
		return "", false
	}
	for _, pAny := range arr {
		p, ok := pAny.(map[string]any)
		if !ok {
			continue
		}
		if p["in"] != "path" {
			continue
		}
		name, _ := p["name"].(string)
		if name != param {
			continue
		}
		s, ok := p["example"].(string)
		return s, ok
	}
	return "", false
}

func walkAugnoLikeStrings(v any, fn func(string)) {
	switch x := v.(type) {
	case string:
		if isAugnoLikeDocID(x) {
			fn(x)
		}
	case map[string]any:
		for _, vv := range x {
			walkAugnoLikeStrings(vv, fn)
		}
	case []any:
		for _, vv := range x {
			walkAugnoLikeStrings(vv, fn)
		}
	}
}
