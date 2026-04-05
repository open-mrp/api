package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	agentdb "github.com/augno/api/services/agent-service/internal/infrastructure/db"
	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"

	"github.com/augno/api/services/agent-service/internal/domain"
)

func HandleSaveMemory(_ context.Context, input json.RawMessage, runCtx *domain.HandlerRunContext) (string, error) {
	var params struct {
		Category   string  `json:"category"`
		Content    string  `json:"content"`
		EntityType string  `json:"entity_type"`
		EntityID   string  `json:"entity_id"`
		Importance float64 `json:"importance"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid save_memory input: %w", err)
	}

	runCtx.Memories = append(runCtx.Memories, domain.PendingMemory{
		Category:   params.Category,
		Content:    params.Content,
		EntityType: params.EntityType,
		EntityID:   params.EntityID,
		Importance: params.Importance,
		Metadata:   input,
	})

	return "Memory saved.", nil
}

func HandleCreateAlert(_ context.Context, input json.RawMessage, runCtx *domain.HandlerRunContext) (string, error) {
	var params struct {
		Severity string `json:"severity"`
		Title    string `json:"title"`
		Message  string `json:"message"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid create_alert input: %w", err)
	}

	runCtx.Alerts = append(runCtx.Alerts, domain.PendingAlert{
		SeverityCode: params.Severity,
		Title:        params.Title,
		Message:      params.Message,
		Metadata:     input,
	})

	return fmt.Sprintf("Alert created: [%s] %s", params.Severity, params.Title), nil
}

func HandleSearchProducts(ctx context.Context, input json.RawMessage, runCtx *domain.HandlerRunContext) (string, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid search_products input: %w", err)
	}
	results, err := runCtx.CoreClient.SearchProducts(ctx, runCtx.AccountID, params.Query)
	if err != nil {
		return "", fmt.Errorf("search_products failed: %w", err)
	}
	if len(results) == 0 {
		return "No products found matching your search.", nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d product(s):\n", len(results))
	for i, p := range results {
		fmt.Fprintf(&sb, "%d. %s (SKU: %s) — $%s [ID: %s]\n", i+1, p.Description, p.SKU, p.UnitPrice, p.ProductID)
	}
	return sb.String(), nil
}

func HandleListProducts(ctx context.Context, _ json.RawMessage, runCtx *domain.HandlerRunContext) (string, error) {
	results, err := runCtx.CoreClient.ListProducts(ctx, runCtx.AccountID)
	if err != nil {
		return "", fmt.Errorf("list_products failed: %w", err)
	}
	if len(results) == 0 {
		return "No products in the catalog.", nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Catalog contains %d product(s):\n", len(results))
	for i, p := range results {
		fmt.Fprintf(&sb, "%d. %s (SKU: %s) — $%s [ID: %s]\n", i+1, p.Description, p.SKU, p.UnitPrice, p.ProductID)
	}
	return sb.String(), nil
}

func HandleLookupCustomer(ctx context.Context, input json.RawMessage, runCtx *domain.HandlerRunContext) (string, error) {
	var params struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid lookup_customer input: %w", err)
	}
	result, err := runCtx.CoreClient.GetCustomerByEmail(ctx, runCtx.AccountID, params.Email)
	if err != nil {
		return "", fmt.Errorf("lookup_customer failed: %w", err)
	}
	if result == nil {
		return "No customer found matching that email address.", nil
	}
	return fmt.Sprintf("Found customer %s (%s) — Relation ID: %s, Role: %s",
		result.UserName, result.Email, result.RelationID, result.RoleCode), nil
}

func HandleCreateArtifact(_ context.Context, input json.RawMessage, runCtx *domain.HandlerRunContext) (string, error) {
	var params struct {
		ArtifactType string `json:"artifact_type"`
		Name         string `json:"name"`
		Content      string `json:"content"`
		MimeType     string `json:"mime_type"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid create_artifact input: %w", err)
	}

	runCtx.Artifacts = append(runCtx.Artifacts, domain.PendingArtifact{
		ArtifactType: params.ArtifactType,
		Name:         params.Name,
		Content:      params.Content,
		MimeType:     params.MimeType,
		Metadata:     input,
	})

	return fmt.Sprintf("Artifact %q created.", params.Name), nil
}

func HandleUpdateMemory(ctx context.Context, input json.RawMessage, runCtx *domain.HandlerRunContext) (string, error) {
	var params struct {
		MemoryID   string  `json:"memory_id"`
		Category   string  `json:"category"`
		Content    string  `json:"content"`
		Importance float64 `json:"importance"`
		EntityType string  `json:"entity_type"`
		EntityID   string  `json:"entity_id"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid update_memory input: %w", err)
	}

	memoryRepo := runCtx.Repos.NewAgentMemoryRepo()

	// Verify ownership
	existing, getErr := memoryRepo.GetByID(ctx, params.MemoryID)
	if getErr != nil {
		return "", fmt.Errorf("update_memory: memory not found: %s", getErr.Error())
	}
	if existing.AccountID != runCtx.AccountID {
		return "", fmt.Errorf("update_memory: memory does not belong to this account")
	}

	if updateErr := memoryRepo.Update(ctx, sqlc.UpdateAgentMemoryParams{
		ID:         params.MemoryID,
		Category:   params.Category,
		Content:    params.Content,
		Metadata:   input,
		EntityType: agentdb.PgText(params.EntityType),
		EntityID:   agentdb.PgText(params.EntityID),
		Importance: params.Importance,
		AccountID:  runCtx.AccountID,
	}); updateErr != nil {
		return "", fmt.Errorf("update_memory failed: %s", updateErr.Error())
	}

	return fmt.Sprintf("Memory %s updated.", params.MemoryID), nil
}

func HandleDeleteMemory(ctx context.Context, input json.RawMessage, runCtx *domain.HandlerRunContext) (string, error) {
	var params struct {
		MemoryID string `json:"memory_id"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid delete_memory input: %w", err)
	}

	memoryRepo := runCtx.Repos.NewAgentMemoryRepo()
	if deleteErr := memoryRepo.Delete(ctx, params.MemoryID, runCtx.AccountID); deleteErr != nil {
		return "", fmt.Errorf("delete_memory failed: %s", deleteErr.Error())
	}

	return fmt.Sprintf("Memory %s deleted.", params.MemoryID), nil
}
