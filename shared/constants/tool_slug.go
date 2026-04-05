package constants

// ToolSlug represents the slug identifier for an agent tool.
type ToolSlug string

const (
	// ToolSlugSaveMemory saves an observation about a customer or product.
	ToolSlugSaveMemory ToolSlug = "save_memory"
	// ToolSlugCreateAlert creates an alert that requires human attention.
	ToolSlugCreateAlert ToolSlug = "create_alert"
	// ToolSlugSearchProducts searches for products by keyword.
	ToolSlugSearchProducts ToolSlug = "search_products"
	// ToolSlugListProducts lists all products in the account catalog.
	ToolSlugListProducts ToolSlug = "list_products"
	// ToolSlugLookupCustomer looks up a customer by email.
	ToolSlugLookupCustomer ToolSlug = "lookup_customer"
	// ToolSlugCreateArtifact creates an artifact (report, document, data export).
	ToolSlugCreateArtifact ToolSlug = "create_artifact"
	// ToolSlugUpdateMemory updates an existing memory entry.
	ToolSlugUpdateMemory ToolSlug = "update_memory"
	// ToolSlugDeleteMemory deletes a memory entry.
	ToolSlugDeleteMemory ToolSlug = "delete_memory"
	// ToolSlugReadDoc reads Augno documentation pages.
	ToolSlugReadDoc ToolSlug = "read_doc"
	// ToolSlugFetchUrl fetches content from a public URL.
	ToolSlugFetchUrl ToolSlug = "fetch_url"
)

func (s ToolSlug) IsValid() bool {
	switch s {
	case ToolSlugSaveMemory, ToolSlugCreateAlert, ToolSlugSearchProducts, ToolSlugListProducts,
		ToolSlugLookupCustomer, ToolSlugCreateArtifact, ToolSlugUpdateMemory, ToolSlugDeleteMemory,
		ToolSlugReadDoc, ToolSlugFetchUrl:
		return true
	default:
		return false
	}
}

func (s ToolSlug) EnumValues() []string {
	return []string{
		string(ToolSlugSaveMemory),
		string(ToolSlugCreateAlert),
		string(ToolSlugSearchProducts),
		string(ToolSlugListProducts),
		string(ToolSlugLookupCustomer),
		string(ToolSlugCreateArtifact),
		string(ToolSlugUpdateMemory),
		string(ToolSlugDeleteMemory),
		string(ToolSlugReadDoc),
		string(ToolSlugFetchUrl),
	}
}
