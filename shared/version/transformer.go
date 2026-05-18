package version

import (
	"slices"

	"github.com/augno/api/shared/constants"
)

// Transformer defines an interface for transforming requests and responses between API versions.
// Each transformer handles the conversion between two adjacent versions:
// - Transform: converts responses from a newer version to an older version (downgrade)
// - TransformRequest: converts requests from an older version to a newer version (upgrade)
type Transformer interface {
	// FromVersion returns the version this transformer applies FROM (newer version)
	FromVersion() APIVersion
	// ToVersion returns the version this transformer applies TO (older version)
	ToVersion() APIVersion
	// ObjectTypes returns the object types this transformer handles
	ObjectTypes() []constants.ObjectType
	// Transform transforms the response data from FromVersion format to ToVersion format (downgrade)
	Transform(objectType constants.ObjectType, data map[string]any) map[string]any
	// TransformRequest transforms the request data from ToVersion format to FromVersion format (upgrade)
	TransformRequest(objectType constants.ObjectType, data map[string]any) map[string]any
}

// TransformerRegistry manages a collection of version transformers.
type TransformerRegistry struct {
	transformers []Transformer
}

// NewTransformerRegistry creates a new empty transformer registry.
func NewTransformerRegistry() *TransformerRegistry {
	return &TransformerRegistry{
		transformers: make([]Transformer, 0),
	}
}

// Register adds a transformer to the registry.
func (r *TransformerRegistry) Register(t Transformer) {
	r.transformers = append(r.transformers, t)
}

// Transform applies the chain of transformers needed to convert response data
// from the 'from' version to the 'to' version for the given object type.
// Transformers are applied in order from newest to oldest (downgrade).
func (r *TransformerRegistry) Transform(from, to APIVersion, objectType constants.ObjectType, data map[string]any) map[string]any {
	if from.Equal(to) || to.After(from) {
		// No transformation needed if versions are equal or requesting newer version
		return data
	}

	result := data

	// Apply transformers in order from 'from' version down to 'to' version
	for _, t := range r.transformers {
		// Check if this transformer applies to our version range
		// The transformer's FromVersion should be <= from and ToVersion should be >= to
		if !t.FromVersion().After(from) && !t.ToVersion().Before(to) {
			// Check if this transformer handles the object type
			if r.handlesObjectType(t, objectType) {
				result = t.Transform(objectType, result)
			}
		}
	}

	return result
}

// TransformRequest applies the chain of transformers needed to convert request data
// from the 'from' version to the 'to' version for the given object type.
// This upgrades requests from older versions to the latest format.
// Transformers are applied in reverse order from oldest to newest (upgrade).
func (r *TransformerRegistry) TransformRequest(from, to APIVersion, objectType constants.ObjectType, data map[string]any) map[string]any {
	if from.Equal(to) || from.After(to) {
		// No transformation needed if versions are equal or already at newer version
		return data
	}

	result := data

	// Apply transformers in reverse order (from 'from' version up to 'to' version)
	// We iterate backwards through the transformer list since transformers are typically
	// registered newest to oldest
	for i := len(r.transformers) - 1; i >= 0; i-- {
		t := r.transformers[i]
		// Check if this transformer applies to our version range
		// For request upgrade: ToVersion should be >= from and FromVersion should be <= to
		if !t.ToVersion().Before(from) && !t.FromVersion().After(to) {
			// Check if this transformer handles the object type
			if r.handlesObjectType(t, objectType) {
				result = t.TransformRequest(objectType, result)
			}
		}
	}

	return result
}

// handlesObjectType checks if the transformer handles the given object type.
func (r *TransformerRegistry) handlesObjectType(t Transformer, objectType constants.ObjectType) bool {
	return slices.Contains(t.ObjectTypes(), objectType)
}

// DefaultRegistry is the global transformer registry.
var DefaultRegistry = NewTransformerRegistry()

// Register adds a transformer to the default registry.
func Register(t Transformer) {
	DefaultRegistry.Register(t)
}

// Transform applies response transformers from the default registry.
func Transform(from, to APIVersion, objectType constants.ObjectType, data map[string]any) map[string]any {
	return DefaultRegistry.Transform(from, to, objectType, data)
}

// TransformRequest applies request transformers from the default registry.
func TransformRequest(from, to APIVersion, objectType constants.ObjectType, data map[string]any) map[string]any {
	return DefaultRegistry.TransformRequest(from, to, objectType, data)
}
