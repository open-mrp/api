package version

import (
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
