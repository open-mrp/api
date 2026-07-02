package agents

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/augno/api/services/agent-service/internal/domain"
)

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
