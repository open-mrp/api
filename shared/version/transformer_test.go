package version

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/open-mrp/api/shared/constants"
)

// mockTransformer implements Transformer for testing
type mockTransformer struct {
	from                 APIVersion
	to                   APIVersion
	objectTypes          []constants.ObjectType
	transformFunc        func(constants.ObjectType, map[string]any) map[string]any
	transformRequestFunc func(constants.ObjectType, map[string]any) map[string]any
}

func (t *mockTransformer) FromVersion() APIVersion             { return t.from }
func (t *mockTransformer) ToVersion() APIVersion               { return t.to }
func (t *mockTransformer) ObjectTypes() []constants.ObjectType { return t.objectTypes }
func (t *mockTransformer) Transform(objectType constants.ObjectType, data map[string]any) map[string]any {
	if t.transformFunc != nil {
		return t.transformFunc(objectType, data)
	}
	return data
}
func (t *mockTransformer) TransformRequest(objectType constants.ObjectType, data map[string]any) map[string]any {
	if t.transformRequestFunc != nil {
		return t.transformRequestFunc(objectType, data)
	}
	return data
}

func TestTransformerRegistry_NoTransformers(t *testing.T) {
	t.Parallel()
	registry := NewTransformerRegistry()

	data := map[string]any{"key": "value"}
	result := registry.Transform(Latest, Latest, constants.ObjectTypeUser, data)

	if !reflect.DeepEqual(result, data) {
		t.Errorf("Expected unchanged data, got %v", result)
	}
}

func TestTransformerRegistry_SameVersion(t *testing.T) {
	t.Parallel()
	registry := NewTransformerRegistry()

	// Register a transformer that would modify data
	registry.Register(&mockTransformer{
		from:        Latest,
		to:          Latest,
		objectTypes: []constants.ObjectType{constants.ObjectTypeUser},
		transformFunc: func(_ constants.ObjectType, data map[string]any) map[string]any {
			data["transformed"] = true
			return data
		},
	})

	data := map[string]any{"key": "value"}
	result := registry.Transform(Latest, Latest, constants.ObjectTypeUser, data)

	// Same version should not transform
	if _, ok := result["transformed"]; ok {
		t.Error("Should not transform when versions are equal")
	}
}

func TestTransformerRegistry_NewerVersionRequested(t *testing.T) {
	t.Parallel()
	older := APIVersion{
		Version:   "1.0.forge",
		Minor:     1,
		Patch:     0,
		Codename:  "forge",
		Preview:   0,
		IsPreview: false,
	}

	newer := APIVersion{
		Version:   "1.1.crucible",
		Minor:     1,
		Patch:     1,
		Codename:  "crucible",
		Preview:   0,
		IsPreview: false,
	}

	registry := NewTransformerRegistry()
	registry.Register(&mockTransformer{
		from:        newer,
		to:          older,
		objectTypes: []constants.ObjectType{constants.ObjectTypeUser},
		transformFunc: func(_ constants.ObjectType, data map[string]any) map[string]any {
			data["transformed"] = true
			return data
		},
	})

	data := map[string]any{"key": "value"}
	// Requesting newer version when we have older should not transform
	result := registry.Transform(older, newer, constants.ObjectTypeUser, data)

	if _, ok := result["transformed"]; ok {
		t.Error("Should not transform when requesting newer version")
	}
}

func TestTransformerRegistry_TransformApplied(t *testing.T) {
	t.Parallel()
	older := APIVersion{
		Version:   "1.0.forge",
		Minor:     1,
		Patch:     0,
		Codename:  "forge",
		Preview:   0,
		IsPreview: false,
	}

	newer := APIVersion{
		Version:   "1.1.crucible",
		Minor:     1,
		Patch:     1,
		Codename:  "crucible",
		Preview:   0,
		IsPreview: false,
	}

	registry := NewTransformerRegistry()
	registry.Register(&mockTransformer{
		from:        newer,
		to:          older,
		objectTypes: []constants.ObjectType{constants.ObjectTypeUser},
		transformFunc: func(_ constants.ObjectType, data map[string]any) map[string]any {
			// Simulate downgrading: rename full_name back to name
			if fullName, ok := data["full_name"]; ok {
				data["name"] = fullName
				delete(data, "full_name")
			}
			return data
		},
	})

	data := map[string]any{"full_name": "John Doe", "email": "john@example.com"}
	result := registry.Transform(newer, older, constants.ObjectTypeUser, data)

	if _, ok := result["full_name"]; ok {
		t.Error("full_name should be removed after transformation")
	}

	if name, ok := result["name"]; !ok || name != "John Doe" {
		t.Errorf("Expected name to be 'John Doe', got %v", result["name"])
	}
}

func TestTransformerRegistry_ObjectTypeMismatch(t *testing.T) {
	t.Parallel()
	older := APIVersion{
		Version:   "1.0.forge",
		Minor:     1,
		Patch:     0,
		Codename:  "forge",
		Preview:   0,
		IsPreview: false,
	}

	newer := APIVersion{
		Version:   "1.1.crucible",
		Minor:     1,
		Patch:     1,
		Codename:  "crucible",
		Preview:   0,
		IsPreview: false,
	}

	registry := NewTransformerRegistry()
	registry.Register(&mockTransformer{
		from:        newer,
		to:          older,
		objectTypes: []constants.ObjectType{constants.ObjectTypeUser},
		transformFunc: func(_ constants.ObjectType, data map[string]any) map[string]any {
			data["transformed"] = true
			return data
		},
	})

	data := map[string]any{"key": "value"}
	// Request transform for "account" but transformer only handles "user"
	result := registry.Transform(newer, older, constants.ObjectTypeAccount, data)

	if _, ok := result["transformed"]; ok {
		t.Error("Should not transform when object type doesn't match")
	}
}

func TestDefaultRegistry(t *testing.T) {
	t.Parallel(
	// Test that default registry exists and can be used
	)

	data := map[string]any{"key": "value"}
	result := Transform(Latest, Latest, constants.ObjectTypeUser, data)

	if !reflect.DeepEqual(result, data) {
		t.Errorf("Expected unchanged data from default registry, got %v", result)
	}
}

func TestTransformerRegistry_MultipleObjectTypes(t *testing.T) {
	t.Parallel()
	older := APIVersion{
		Version:   "1.0.forge",
		Minor:     1,
		Patch:     0,
		Codename:  "forge",
		Preview:   0,
		IsPreview: false,
	}

	newer := APIVersion{
		Version:   "1.1.crucible",
		Minor:     1,
		Patch:     1,
		Codename:  "crucible",
		Preview:   0,
		IsPreview: false,
	}

	registry := NewTransformerRegistry()
	registry.Register(&mockTransformer{
		from:        newer,
		to:          older,
		objectTypes: []constants.ObjectType{constants.ObjectTypeUser, constants.ObjectTypeAccount},
		transformFunc: func(_ constants.ObjectType, data map[string]any) map[string]any {
			data["transformed"] = true
			return data
		},
	})

	// Should work for both user and account
	for _, objectType := range []constants.ObjectType{constants.ObjectTypeUser, constants.ObjectTypeAccount} {
		data := map[string]any{"key": "value"}
		result := registry.Transform(newer, older, objectType, data)

		if _, ok := result["transformed"]; !ok {
			t.Errorf("Expected transform to apply for object type %s", objectType)
		}
	}
}

func TestTransformerRegistry_TransformRequestApplied(t *testing.T) {
	t.Parallel()
	older := APIVersion{
		Version:   "1.0.forge",
		Minor:     1,
		Patch:     0,
		Codename:  "forge",
		Preview:   0,
		IsPreview: false,
	}

	newer := APIVersion{
		Version:   "1.1.crucible",
		Minor:     1,
		Patch:     1,
		Codename:  "crucible",
		Preview:   0,
		IsPreview: false,
	}

	registry := NewTransformerRegistry()
	registry.Register(&mockTransformer{
		from:        newer,
		to:          older,
		objectTypes: []constants.ObjectType{constants.ObjectTypeUser},
		transformRequestFunc: func(_ constants.ObjectType, data map[string]any) map[string]any {
			// Simulate upgrading: rename name to full_name
			if name, ok := data["name"]; ok {
				data["full_name"] = name
				delete(data, "name")
			}
			return data
		},
	})

	// Request with old field "name" should be upgraded to "full_name"
	data := map[string]any{"name": "John Doe", "email": "john@example.com"}
	result := registry.TransformRequest(older, newer, constants.ObjectTypeUser, data)

	if _, ok := result["name"]; ok {
		t.Error("name should be removed after request transformation")
	}

	if fullName, ok := result["full_name"]; !ok || fullName != "John Doe" {
		t.Errorf("Expected full_name to be 'John Doe', got %v", result["full_name"])
	}
}

func TestTransformerRegistry_TransformRequestSameVersion(t *testing.T) {
	t.Parallel()
	registry := NewTransformerRegistry()

	registry.Register(&mockTransformer{
		from:        Latest,
		to:          Latest,
		objectTypes: []constants.ObjectType{constants.ObjectTypeUser},
		transformRequestFunc: func(_ constants.ObjectType, data map[string]any) map[string]any {
			data["transformed"] = true
			return data
		},
	})

	data := map[string]any{"key": "value"}
	result := registry.TransformRequest(Latest, Latest, constants.ObjectTypeUser, data)

	// Same version should not transform
	if _, ok := result["transformed"]; ok {
		t.Error("Should not transform request when versions are equal")
	}
}

func TestTransformerRegistry_TransformRequestOlderVersionRequested(t *testing.T) {
	t.Parallel()
	older := APIVersion{
		Version:   "1.0.forge",
		Minor:     1,
		Patch:     0,
		Codename:  "forge",
		Preview:   0,
		IsPreview: false,
	}

	newer := APIVersion{
		Version:   "1.1.crucible",
		Minor:     1,
		Patch:     1,
		Codename:  "crucible",
		Preview:   0,
		IsPreview: false,
	}

	registry := NewTransformerRegistry()
	registry.Register(&mockTransformer{
		from:        newer,
		to:          older,
		objectTypes: []constants.ObjectType{constants.ObjectTypeUser},
		transformRequestFunc: func(_ constants.ObjectType, data map[string]any) map[string]any {
			data["transformed"] = true
			return data
		},
	})

	data := map[string]any{"key": "value"}
	// Requesting to transform from newer to older (downgrade) should not apply TransformRequest
	result := registry.TransformRequest(newer, older, constants.ObjectTypeUser, data)

	if _, ok := result["transformed"]; ok {
		t.Error("Should not transform request when going from newer to older version")
	}
}

func TestTransformerRegistry_BidirectionalTransformation(t *testing.T) {
	t.Parallel()
	older := APIVersion{
		Version:   "1.0.forge",
		Minor:     1,
		Patch:     0,
		Codename:  "forge",
		Preview:   0,
		IsPreview: false,
	}

	newer := APIVersion{
		Version:   "1.1.crucible",
		Minor:     1,
		Patch:     1,
		Codename:  "crucible",
		Preview:   0,
		IsPreview: false,
	}

	registry := NewTransformerRegistry()
	registry.Register(&mockTransformer{
		from:        newer,
		to:          older,
		objectTypes: []constants.ObjectType{constants.ObjectTypeUser},
		// Response downgrade: full_name -> name
		transformFunc: func(_ constants.ObjectType, data map[string]any) map[string]any {
			if fullName, ok := data["full_name"]; ok {
				data["name"] = fullName
				delete(data, "full_name")
			}
			return data
		},
		// Request upgrade: name -> full_name
		transformRequestFunc: func(_ constants.ObjectType, data map[string]any) map[string]any {
			if name, ok := data["name"]; ok {
				data["full_name"] = name
				delete(data, "name")
			}
			return data
		},
	})

	// Test request upgrade (older -> newer)
	reqData := map[string]any{"name": "Jane Doe", "email": "jane@example.com"}
	upgradedReq := registry.TransformRequest(older, newer, constants.ObjectTypeUser, reqData)

	if _, ok := upgradedReq["name"]; ok {
		t.Error("Request should have name removed after upgrade")
	}
	if fullName, ok := upgradedReq["full_name"]; !ok || fullName != "Jane Doe" {
		t.Errorf("Request should have full_name='Jane Doe' after upgrade, got %v", upgradedReq["full_name"])
	}

	// Test response downgrade (newer -> older)
	respData := map[string]any{"full_name": "Jane Doe", "email": "jane@example.com"}
	downgradedResp := registry.Transform(newer, older, constants.ObjectTypeUser, respData)

	if _, ok := downgradedResp["full_name"]; ok {
		t.Error("Response should have full_name removed after downgrade")
	}
	if name, ok := downgradedResp["name"]; !ok || name != "Jane Doe" {
		t.Errorf("Response should have name='Jane Doe' after downgrade, got %v", downgradedResp["name"])
	}
}

func TestDefaultRegistry_TransformRequest(t *testing.T) {
	t.Parallel(
	// Test that default registry's TransformRequest exists and can be used
	)

	data := map[string]any{"key": "value"}
	result := TransformRequest(Latest, Latest, constants.ObjectTypeUser, data)

	if !reflect.DeepEqual(result, data) {
		t.Errorf("Expected unchanged data from default registry TransformRequest, got %v", result)
	}
}

// mockIncludeForcer is a mockTransformer that also declares forced includes.
type mockIncludeForcer struct {
	mockTransformer
	forcedIncludes func(constants.ObjectType) []string
}

func (t *mockIncludeForcer) ForcedIncludes(objectType constants.ObjectType) []string {
	if t.forcedIncludes != nil {
		return t.forcedIncludes(objectType)
	}
	return nil
}

func previewVersion(preview int) APIVersion {
	return APIVersion{
		Version:   fmt.Sprintf("1.0.forge-preview.%d", preview),
		Minor:     1,
		Patch:     0,
		Codename:  "forge",
		Preview:   preview,
		IsPreview: true,
	}
}

// recordingTransformer appends its name to order and stamps the payload, so a chain's execution sequence is observable.
func recordingTransformer(order *[]string, name string, from, to APIVersion, objectTypes ...constants.ObjectType) *mockTransformer {
	stamp := func(_ constants.ObjectType, data map[string]any) map[string]any {
		*order = append(*order, name)
		if data != nil {
			data["steps"] = append(data["steps"].([]string), name)
		}
		return data
	}

	return &mockTransformer{
		from:                 from,
		to:                   to,
		objectTypes:          objectTypes,
		transformFunc:        stamp,
		transformRequestFunc: stamp,
	}
}

// The registry walks its transformers in registration order, so a downgrade chain only runs newest-to-oldest when it was registered in that order.
func TestTransformerRegistry_TransformChainNewestToOldest(t *testing.T) {
	t.Parallel()
	v1, v2, v3 := previewVersion(1), previewVersion(2), previewVersion(3)

	var order []string
	registry := NewTransformerRegistry()
	registry.Register(recordingTransformer(&order, "3->2", v3, v2, constants.ObjectTypeUser))
	registry.Register(recordingTransformer(&order, "2->1", v2, v1, constants.ObjectTypeUser))

	data := map[string]any{"steps": []string{}}
	result := registry.Transform(v3, v1, constants.ObjectTypeUser, data)

	want := []string{"3->2", "2->1"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("Expected downgrade steps to run %v, got %v", want, order)
	}
	if steps, ok := result["steps"].([]string); !ok || !reflect.DeepEqual(steps, want) {
		t.Errorf("Expected payload to record steps %v, got %v", want, result["steps"])
	}
}

// Mirror of the downgrade chain: the reverse walk yields oldest-to-newest only for a newest-to-oldest registration order.
func TestTransformerRegistry_TransformRequestChainOldestToNewest(t *testing.T) {
	t.Parallel()
	v1, v2, v3 := previewVersion(1), previewVersion(2), previewVersion(3)

	var order []string
	registry := NewTransformerRegistry()
	registry.Register(recordingTransformer(&order, "3->2", v3, v2, constants.ObjectTypeUser))
	registry.Register(recordingTransformer(&order, "2->1", v2, v1, constants.ObjectTypeUser))

	data := map[string]any{"steps": []string{}}
	result := registry.TransformRequest(v1, v3, constants.ObjectTypeUser, data)

	want := []string{"2->1", "3->2"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("Expected upgrade steps to run %v, got %v", want, order)
	}
	if steps, ok := result["steps"].([]string); !ok || !reflect.DeepEqual(steps, want) {
		t.Errorf("Expected payload to record steps %v, got %v", want, result["steps"])
	}
}

// The range predicate is the only thing keeping a transformer away from a payload whose versions it knows nothing about.
func TestTransformerRegistry_TransformSkipsOutOfRangeTransformers(t *testing.T) {
	t.Parallel()
	v1, v2, v3, v4 := previewVersion(1), previewVersion(2), previewVersion(3), previewVersion(4)

	var order []string
	registry := NewTransformerRegistry()
	registry.Register(recordingTransformer(&order, "4->3", v4, v3, constants.ObjectTypeUser))
	registry.Register(recordingTransformer(&order, "2->1", v2, v1, constants.ObjectTypeUser))

	data := map[string]any{"steps": []string{}}
	registry.Transform(v3, v2, constants.ObjectTypeUser, data)

	if len(order) != 0 {
		t.Errorf("Expected no transformer to run for a 3 -> 2 downgrade, got %v", order)
	}
}

func TestTransformerRegistry_TransformRequestSkipsOutOfRangeTransformers(t *testing.T) {
	t.Parallel()
	v1, v2, v3, v4 := previewVersion(1), previewVersion(2), previewVersion(3), previewVersion(4)

	var order []string
	registry := NewTransformerRegistry()
	registry.Register(recordingTransformer(&order, "4->3", v4, v3, constants.ObjectTypeUser))
	registry.Register(recordingTransformer(&order, "2->1", v2, v1, constants.ObjectTypeUser))

	data := map[string]any{"steps": []string{}}
	registry.TransformRequest(v2, v3, constants.ObjectTypeUser, data)

	if len(order) != 0 {
		t.Errorf("Expected no transformer to run for a 2 -> 3 upgrade, got %v", order)
	}
}

// A handler response that marshals to JSON null reaches the registry as a nil map; transformers must be handed it as-is rather than the registry substituting something.
func TestTransformerRegistry_NilAndEmptyData(t *testing.T) {
	t.Parallel()
	older, newer := previewVersion(1), previewVersion(2)

	tests := []struct {
		name string
		data map[string]any
	}{
		{"nil", nil},
		{"empty", map[string]any{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sawNil, called bool
			registry := NewTransformerRegistry()
			registry.Register(&mockTransformer{
				from:        newer,
				to:          older,
				objectTypes: []constants.ObjectType{constants.ObjectTypeUser},
				transformFunc: func(_ constants.ObjectType, data map[string]any) map[string]any {
					called = true
					sawNil = data == nil
					return data
				},
			})

			result := registry.Transform(newer, older, constants.ObjectTypeUser, tt.data)

			if !called {
				t.Fatal("Expected transformer to be invoked")
			}
			if sawNil != (tt.data == nil) {
				t.Errorf("Expected transformer to receive nil=%t, got nil=%t", tt.data == nil, sawNil)
			}
			if len(result) != 0 {
				t.Errorf("Expected empty result, got %v", result)
			}
		})
	}
}

func TestTransformerRegistry_TransformerReturningNil(t *testing.T) {
	t.Parallel()
	v1, v2, v3 := previewVersion(1), previewVersion(2), previewVersion(3)

	registry := NewTransformerRegistry()
	registry.Register(&mockTransformer{
		from:        v3,
		to:          v2,
		objectTypes: []constants.ObjectType{constants.ObjectTypeUser},
		transformFunc: func(_ constants.ObjectType, _ map[string]any) map[string]any {
			return nil
		},
	})
	registry.Register(&mockTransformer{
		from:        v2,
		to:          v1,
		objectTypes: []constants.ObjectType{constants.ObjectTypeUser},
		transformFunc: func(_ constants.ObjectType, data map[string]any) map[string]any {
			if data != nil {
				t.Error("Expected the nil returned by the previous step to be passed through")
			}
			return data
		},
	})

	result := registry.Transform(v3, v1, constants.ObjectTypeUser, map[string]any{"key": "value"})

	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}

// --- ForcedIncludes tests ---

func TestTransformerRegistry_ForcedIncludes_TransformerWithoutForcer(t *testing.T) {
	t.Parallel()
	older, newer := previewVersion(1), previewVersion(2)

	registry := NewTransformerRegistry()
	registry.Register(&mockTransformer{
		from:        newer,
		to:          older,
		objectTypes: []constants.ObjectType{constants.ObjectTypeUser},
	})

	keys := registry.ForcedIncludes(newer, older, constants.ObjectTypeUser)
	if len(keys) != 0 {
		t.Errorf("Expected no forced includes from a transformer that is not an IncludeForcer, got %v", keys)
	}
}

func TestTransformerRegistry_ForcedIncludes_Deduplicates(t *testing.T) {
	t.Parallel()
	v1, v2, v3 := previewVersion(1), previewVersion(2), previewVersion(3)

	forcer := func(from, to APIVersion, keys ...string) *mockIncludeForcer {
		return &mockIncludeForcer{
			mockTransformer: mockTransformer{
				from:        from,
				to:          to,
				objectTypes: []constants.ObjectType{constants.ObjectTypeUser},
			},
			forcedIncludes: func(constants.ObjectType) []string { return keys },
		}
	}

	registry := NewTransformerRegistry()
	registry.Register(forcer(v3, v2, "user", "user.account"))
	registry.Register(forcer(v2, v1, "user", "user.address"))

	keys := registry.ForcedIncludes(v3, v1, constants.ObjectTypeUser)

	want := []string{"user", "user.account", "user.address"}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("Expected deduplicated includes %v, got %v", want, keys)
	}
}

func TestTransformerRegistry_ForcedIncludes_NoDowngrade(t *testing.T) {
	t.Parallel()
	older, newer := previewVersion(1), previewVersion(2)

	registry := NewTransformerRegistry()
	registry.Register(&mockIncludeForcer{
		mockTransformer: mockTransformer{
			from:        newer,
			to:          older,
			objectTypes: []constants.ObjectType{constants.ObjectTypeUser},
		},
		forcedIncludes: func(constants.ObjectType) []string { return []string{"user"} },
	})

	tests := []struct {
		name     string
		from, to APIVersion
	}{
		{"equal versions", newer, newer},
		{"requesting a newer version", older, newer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := registry.ForcedIncludes(tt.from, tt.to, constants.ObjectTypeUser)
			if keys != nil {
				t.Errorf("Expected nil forced includes, got %v", keys)
			}
		})
	}
}

func TestTransformerRegistry_ForcedIncludes_OutOfRangeOrWrongObjectType(t *testing.T) {
	t.Parallel()
	v1, v2, v3, v4 := previewVersion(1), previewVersion(2), previewVersion(3), previewVersion(4)

	registry := NewTransformerRegistry()
	registry.Register(&mockIncludeForcer{
		mockTransformer: mockTransformer{
			from:        v4,
			to:          v3,
			objectTypes: []constants.ObjectType{constants.ObjectTypeUser},
		},
		forcedIncludes: func(constants.ObjectType) []string { return []string{"out-of-range"} },
	})
	registry.Register(&mockIncludeForcer{
		mockTransformer: mockTransformer{
			from:        v2,
			to:          v1,
			objectTypes: []constants.ObjectType{constants.ObjectTypeAccount},
		},
		forcedIncludes: func(constants.ObjectType) []string { return []string{"wrong-object-type"} },
	})
	registry.Register(&mockIncludeForcer{
		mockTransformer: mockTransformer{
			from:        v2,
			to:          v1,
			objectTypes: []constants.ObjectType{constants.ObjectTypeUser},
		},
		forcedIncludes: func(constants.ObjectType) []string { return []string{"user"} },
	})

	keys := registry.ForcedIncludes(v2, v1, constants.ObjectTypeUser)

	want := []string{"user"}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("Expected only the in-range includes %v, got %v", want, keys)
	}
}

func TestTransformerRegistry_ForcedIncludes_ObjectTypeScoped(t *testing.T) {
	t.Parallel()
	older, newer := previewVersion(1), previewVersion(2)

	registry := NewTransformerRegistry()
	registry.Register(&mockIncludeForcer{
		mockTransformer: mockTransformer{
			from:        newer,
			to:          older,
			objectTypes: []constants.ObjectType{constants.ObjectTypeUser, constants.ObjectTypeAccount},
		},
		forcedIncludes: func(objectType constants.ObjectType) []string {
			if objectType == constants.ObjectTypeAccount {
				return nil
			}
			return []string{"user.account"}
		},
	})

	if keys := registry.ForcedIncludes(newer, older, constants.ObjectTypeAccount); len(keys) != 0 {
		t.Errorf("Expected no forced includes for account, got %v", keys)
	}

	want := []string{"user.account"}
	if keys := registry.ForcedIncludes(newer, older, constants.ObjectTypeUser); !reflect.DeepEqual(keys, want) {
		t.Errorf("Expected forced includes %v for user, got %v", want, keys)
	}
}

func TestDefaultRegistry_ForcedIncludes(t *testing.T) {
	t.Parallel()
	if keys := ForcedIncludes(Latest, Latest, constants.ObjectTypeUser); keys != nil {
		t.Errorf("Expected nil forced includes from default registry for equal versions, got %v", keys)
	}
}
