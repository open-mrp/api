package agents

import "github.com/augno/api/services/agent-service/internal/domain"

func RegisterTools(registry *domain.ToolHandlerRegistry) {
	registry.Register("save_memory", HandleSaveMemory)
	registry.Register("create_alert", HandleCreateAlert)
	registry.Register("search_products", HandleSearchProducts)
	registry.Register("list_products", HandleListProducts)
	registry.Register("lookup_customer", HandleLookupCustomer)
	registry.Register("create_artifact", HandleCreateArtifact)
	registry.Register("update_memory", HandleUpdateMemory)
	registry.Register("delete_memory", HandleDeleteMemory)
	registry.Register("read_doc", HandleReadDoc)
	registry.Register("fetch_url", HandleFetchURL)
}
