// Package versiontransforms registers the transformers that serve 1.0.forge-preview.9 callers from the 1.0.forge
// backend: responses are downgraded to the preview.9 shape and preview.9 requests are upgraded. Each file is named
// transform_<resource>_forge_to_forge_preview_9.go, and its init registers it.
package versiontransforms
