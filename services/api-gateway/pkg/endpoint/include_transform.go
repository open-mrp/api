package apiendpoint

import (
	"strings"

	"github.com/augno/api/shared/constants"
)

// CollapseUnexpanded replaces non-requested include fields with null.
// It handles both single-object and list responses (object == "list").
func CollapseUnexpanded(data map[string]any, config *IncludeConfig, requested map[string]bool) map[string]any {
	if config == nil {
		return data
	}

	fields := config.FieldsByKey()

	// List response: iterate over data[] items.
	if obj, ok := data["object"]; ok && obj == string(constants.ObjectTypeList) {
		if items, ok := data["data"].([]any); ok {
			for i, item := range items {
				if m, ok := item.(map[string]any); ok {
					items[i] = collapseFields(m, fields, requested)
				}
			}
		}
		return data
	}

	return collapseFields(data, fields, requested)
}

// collapseFields collapses all non-requested include fields on a single object.
// A parent field (e.g. "actor") is kept when a child include (e.g. "actor.role")
// is requested, so that the nested expansion remains reachable.
func collapseFields(data map[string]any, fields map[string]IncludeField, requested map[string]bool) map[string]any {
	for key, field := range fields {
		if requested[key] || hasChildInclude(key, requested) {
			continue
		}
		for _, path := range field.JSONPaths {
			collapseAtPath(data, path)
		}
	}
	return data
}

// hasChildInclude returns true when any requested include key is a child of
// the given parent (e.g. parent="actor" matches requested key "actor.role").
func hasChildInclude(parent string, requested map[string]bool) bool {
	prefix := parent + "."
	for key := range requested {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// collapseAtPath navigates a dot-separated JSON path and sets the terminal
// value to nil. Handles nil/missing values gracefully.
func collapseAtPath(data map[string]any, path string) {
	parts := strings.Split(path, ".")
	current := data

	// Navigate to the parent of the terminal key.
	for _, part := range parts[:len(parts)-1] {
		child, ok := current[part]
		if !ok || child == nil {
			return
		}
		m, ok := child.(map[string]any)
		if !ok {
			return
		}
		current = m
	}

	terminalKey := parts[len(parts)-1]
	val, ok := current[terminalKey]
	if !ok || val == nil {
		return
	}

	current[terminalKey] = nil
}
