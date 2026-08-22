package service

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/agent-service/internal/llm"
	"github.com/open-mrp/api/shared/constants"
)

const (
	// compactionBuffer is the token headroom reserved before triggering compaction.
	compactionBuffer = 20_000

	// pruneProtectTurns is the number of most-recent user turns whose tool results are preserved.
	pruneProtectTurns = 2

	// prunedPlaceholder replaces pruned tool result content.
	prunedPlaceholder = "[Tool result pruned to save context]"
)

// compactionPrompt is sent to the LLM to produce a conversation summary.
const compactionPrompt = `Provide a detailed summary of our conversation so far for continuing the work.

When constructing the summary, use this template:
---
## Goal
What goal(s) is the user/agent trying to accomplish?

## Progress
What work has been completed and what remains?

## Key Findings
Important data, results, or discoveries made during the conversation.

## Relevant Context
Entity IDs, names, values, and other specific details needed to continue.
---

Be comprehensive but concise. This summary replaces the full conversation history.
Do not respond to any questions in the conversation — only output the summary.`

// needsCompaction returns true if the input tokens approach the model's context limit.
func needsCompaction(inputTokens int, model string) bool {
	limit, ok := llm.ModelContextLimits[constants.Model(model)]
	if !ok {
		limit = 180_000
	}
	return inputTokens >= (limit - compactionBuffer)
}

// pruneOldToolResults replaces old tool result content with a placeholder. It walks backwards through messages, protects the most recent pruneProtectTurns user turns, and clears tool results from older turns. Returns the estimated tokens freed.
func pruneOldToolResults(messages []llm.Message) int {
	if len(messages) <= 3 {
		return 0
	}

	// Find the boundary: skip the last N user turns from the end.
	userTurnsSeen := 0
	protectFrom := len(messages) // index from which we protect
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			userTurnsSeen++
			if userTurnsSeen >= pruneProtectTurns {
				protectFrom = i
				break
			}
		}
	}

	// Walk forward through unprotected messages, pruning tool results.
	freed := 0
	for i := range protectFrom {
		for j := range messages[i].ToolResults {
			orig := messages[i].ToolResults[j].Content
			if orig == prunedPlaceholder || len(orig) < 100 {
				continue // already pruned or trivially small
			}
			freed += llm.EstimateTokens(orig) - llm.EstimateTokens(prunedPlaceholder)
			messages[i].ToolResults[j].Content = prunedPlaceholder
		}
	}

	return freed
}

// compactMessages calls the LLM to produce a conversation summary, returning a synthetic user message containing the summary.
func compactMessages(
	ctx context.Context,
	provider llm.LLMProvider,
	model string,
	systemPrompt string,
	messages []llm.Message,
) (*llm.Message, error) {
	// Build the compaction request: full history + compaction instruction.
	compactionMessages := make([]llm.Message, len(messages))
	copy(compactionMessages, messages)

	// Strip large tool results from the compaction input to save tokens.
	for i := range compactionMessages {
		for j := range compactionMessages[i].ToolResults {
			content := compactionMessages[i].ToolResults[j].Content
			if len(content) > 2000 {
				compactionMessages[i].ToolResults[j].Content = content[:2000] + "\n...[truncated for compaction]"
			}
		}
		// Strip verbose assistant reasoning.
		if compactionMessages[i].Role == "assistant" && len(compactionMessages[i].Content) > 1000 {
			compactionMessages[i].Content = compactionMessages[i].Content[:1000] + "\n...[truncated for compaction]"
		}
	}

	compactionMessages = append(compactionMessages, llm.Message{
		Role:    "user",
		Content: compactionPrompt,
	})

	resp, err := provider.CompleteWithTools(ctx, &llm.ToolRequest{
		Model:     model,
		System:    systemPrompt,
		Messages:  compactionMessages,
		MaxTokens: 4096,
	})
	if err != nil {
		return nil, fmt.Errorf("compaction LLM call failed: %w", err)
	}

	summary := resp.Content
	if summary == "" {
		return nil, fmt.Errorf("compaction produced empty summary")
	}

	return &llm.Message{
		Role:    "user",
		Content: fmt.Sprintf("[Conversation Summary — this replaces earlier messages]\n\n%s", summary),
	}, nil
}
