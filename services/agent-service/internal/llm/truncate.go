package llm

import (
	"fmt"

	"github.com/augno/api/shared/constants"
)

// ModelLimits defines the context window and output token reservation per model.
type ModelLimits struct {
	ContextLimit  int // maximum prompt tokens
	OutputReserve int // tokens reserved for output generation
}

// ModelContextLimits defines the maximum prompt token budget per model. These are set conservatively below the absolute API limits to leave headroom for token-estimation inaccuracy and output tokens.
var ModelContextLimits = map[constants.Model]int{
	constants.ModelClaudeOpus48:   180000,
	constants.ModelClaudeSonnet46: 180000,
	constants.ModelClaudeSonnet4:  180000,
	constants.ModelClaudeHaiku45:  180000,
	constants.ModelGPT55:          115000,
	constants.ModelGPT4o:          115000,
	constants.ModelGPT4oMini:      115000,
}

// ModelLimitsMap provides structured limits per model, including output reservation.
var ModelLimitsMap = map[constants.Model]ModelLimits{
	constants.ModelClaudeOpus48:   {ContextLimit: 180000, OutputReserve: 8192},
	constants.ModelClaudeSonnet46: {ContextLimit: 180000, OutputReserve: 8192},
	constants.ModelClaudeSonnet4:  {ContextLimit: 180000, OutputReserve: 8192},
	constants.ModelClaudeHaiku45:  {ContextLimit: 180000, OutputReserve: 8192},
	constants.ModelGPT55:          {ContextLimit: 115000, OutputReserve: 4096},
	constants.ModelGPT4o:          {ContextLimit: 115000, OutputReserve: 4096},
	constants.ModelGPT4oMini:      {ContextLimit: 115000, OutputReserve: 4096},
}

// GetModelLimits returns the ModelLimits for the given model, falling back to defaults.
func GetModelLimits(model string) ModelLimits {
	if ml, ok := ModelLimitsMap[constants.Model(model)]; ok {
		return ml
	}
	return ModelLimits{ContextLimit: defaultContextLimit, OutputReserve: 4096}
}

// EstimateContextUsage estimates the total token usage for a set of messages, system prompt, and tool definitions.
func EstimateContextUsage(system string, messages []Message, tools []ToolDefinition) int {
	return EstimateTokens(system) + estimateToolDefsTokens(tools) + EstimateAllMessages(messages)
}

// ContextBudgetRemaining returns the estimated tokens remaining before hitting the proactive compaction threshold (85% of context limit).
func ContextBudgetRemaining(system string, messages []Message, tools []ToolDefinition, model string) int {
	ml := GetModelLimits(model)
	threshold := int(float64(ml.ContextLimit) * 0.85)
	used := EstimateContextUsage(system, messages, tools)
	remaining := threshold - used
	if remaining < 0 {
		return 0
	}
	return remaining
}

// NeedsProactiveCompaction returns true if the estimated token usage exceeds 85% of the model's context limit, indicating compaction should be triggered before the next LLM call to avoid a hard context overflow error.
func NeedsProactiveCompaction(system string, messages []Message, tools []ToolDefinition, model string) bool {
	return ContextBudgetRemaining(system, messages, tools, model) == 0
}

const (
	defaultContextLimit = 180000

	// Truncation thresholds applied in order of priority.
	maxAssistantContentChars = 2000  // assistant thinking/reasoning — most expendable
	maxToolResultCharsRecent = 20000 // tool results from recent turns (last 2 user turns)
	maxToolResultCharsOld    = 8000  // tool results from older turns
	maxToolInputChars        = 5000  // tool call input JSON
	truncationHeadPreserve   = 500   // chars preserved from start when truncating tool results
	truncationTailPreserve   = 200   // chars preserved from end when truncating tool results

	truncationPlaceholder = "[Earlier conversation history was removed to fit within the context window. The most recent messages are preserved below.]"
)

// EstimateTokens returns a rough token count for a string. Uses ~4 characters per token, which is conservative for English/code.
func EstimateTokens(s string) int {
	n := len(s)
	if n == 0 {
		return 0
	}
	return (n + 3) / 4 // ceiling division
}

// estimateMessageTokens estimates the token count for a single message.
func estimateMessageTokens(m Message) int {
	tokens := 4 // role + message overhead
	tokens += EstimateTokens(m.Content)
	for _, tu := range m.ToolUse {
		tokens += 4 // tool use overhead
		tokens += EstimateTokens(tu.Name)
		tokens += EstimateTokens(string(tu.Input))
	}
	for _, tr := range m.ToolResults {
		tokens += 4
		tokens += EstimateTokens(tr.Content)
	}
	return tokens
}

// estimateToolDefsTokens estimates token overhead for tool definitions.
func estimateToolDefsTokens(tools []ToolDefinition) int {
	tokens := 0
	for _, t := range tools {
		tokens += 10 // per-tool overhead
		tokens += EstimateTokens(t.Name)
		tokens += EstimateTokens(t.Description)
		tokens += EstimateTokens(string(t.InputSchema))
	}
	return tokens
}

// TruncateMessages ensures the total prompt fits within the model's context limit.
//
// Truncation is applied in priority order to minimize information loss:
//  1. Truncate assistant message content (thinking/reasoning — most expendable)
//  2. Cap oversized tool result content
//  3. Cap oversized tool call inputs
//  4. Drop old assistant+tool messages from the middle (user messages preserved)
//  5. Drop old messages from the middle as a last resort
func TruncateMessages(system string, messages []Message, tools []ToolDefinition, model string) []Message {
	limit, ok := ModelContextLimits[constants.Model(model)]
	if !ok {
		limit = defaultContextLimit
	}

	// Calculate fixed overhead (system prompt + tool definitions + output buffer).
	fixedTokens := EstimateTokens(system) + estimateToolDefsTokens(tools) + 4096

	budget := limit - fixedTokens
	if budget <= 0 {
		return messages
	}

	// If everything fits already, return as-is.
	if EstimateAllMessages(messages) <= budget {
		return messages
	}

	// Deep copy so we don't mutate the caller's slice.
	messages = CopyMessages(messages)

	// Pass 1: Truncate assistant content (thinking/reasoning).
	truncateAssistantContent(messages)
	if EstimateAllMessages(messages) <= budget {
		return messages
	}

	// Pass 2: Cap oversized tool results.
	capToolResults(messages)
	if EstimateAllMessages(messages) <= budget {
		return messages
	}

	// Pass 3: Cap oversized tool call inputs.
	capToolInputs(messages)
	if EstimateAllMessages(messages) <= budget {
		return messages
	}

	// Pass 4: Drop old non-user messages from the middle.
	messages = dropOldNonUserMessages(messages, budget)
	if EstimateAllMessages(messages) <= budget {
		return messages
	}

	// Pass 5: Drop old messages from the middle (last resort).
	return dropOldMessages(messages, budget)
}

// truncateAssistantContent caps the text content of assistant messages. This targets verbose reasoning/thinking that the LLM produced but doesn't need to re-read in full to continue the conversation.
func truncateAssistantContent(messages []Message) {
	for i := range messages {
		if messages[i].Role != "assistant" {
			continue
		}
		if len(messages[i].Content) > maxAssistantContentChars {
			messages[i].Content = messages[i].Content[:maxAssistantContentChars] +
				fmt.Sprintf("\n... [assistant reasoning truncated: %d chars removed]",
					len(messages[i].Content)-maxAssistantContentChars)
		}
	}
}

func capToolResults(messages []Message) {
	// Find the boundary for "recent" messages: last 2 user turns.
	recentFrom := len(messages)
	userTurnsSeen := 0
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			userTurnsSeen++
			if userTurnsSeen >= 2 {
				recentFrom = i
				break
			}
		}
	}

	for i := range messages {
		maxChars := maxToolResultCharsOld
		if i >= recentFrom {
			maxChars = maxToolResultCharsRecent
		}
		for j := range messages[i].ToolResults {
			orig := messages[i].ToolResults[j].Content
			if len(orig) <= maxChars {
				continue
			}
			// Preserve head and tail for better context.
			head := orig[:truncationHeadPreserve]
			tail := ""
			if len(orig) > truncationTailPreserve {
				tail = orig[len(orig)-truncationTailPreserve:]
			}
			messages[i].ToolResults[j].Content = fmt.Sprintf(
				"%s\n... [truncated: %d chars removed] ...\n%s",
				head, len(orig)-truncationHeadPreserve-len(tail), tail)
		}
	}
}

func capToolInputs(messages []Message) {
	for i := range messages {
		for j := range messages[i].ToolUse {
			orig := string(messages[i].ToolUse[j].Input)
			if len(orig) > maxToolInputChars {
				messages[i].ToolUse[j].Input = []byte(
					orig[:maxToolInputChars] +
						fmt.Sprintf("... [truncated: %d chars total]", len(orig)))
			}
		}
	}
}

// dropOldNonUserMessages removes assistant and tool-result messages from the middle of the conversation, oldest first, while preserving all user messages. This keeps the user's intent and inputs intact.
func dropOldNonUserMessages(messages []Message, budget int) []Message {
	if len(messages) <= 3 {
		return messages
	}

	// We always keep the first message and the last 4 messages.
	keepTail := min(4, len(messages)-1)
	protected := len(messages) - keepTail

	// Mark non-user messages in the middle for removal, oldest first.
	drop := make([]bool, len(messages))
	total := EstimateAllMessages(messages)

	for i := 1; i < protected && total > budget; i++ {
		if messages[i].Role == "user" && len(messages[i].ToolResults) == 0 {
			// Pure user message — skip (preserve user intent).
			continue
		}
		total -= estimateMessageTokens(messages[i])
		drop[i] = true
	}

	return collectMessages(messages, drop)
}

// dropOldMessages drops messages from the middle of the conversation as a last resort, keeping the first message and as many recent messages as fit.
func dropOldMessages(messages []Message, budget int) []Message {
	if len(messages) <= 2 {
		return messages
	}

	firstTokens := estimateMessageTokens(messages[0])
	placeholderTokens := EstimateTokens(truncationPlaceholder) + 8
	remaining := budget - firstTokens - placeholderTokens

	if remaining <= 0 {
		// Can't even fit the first message; keep only what fits from the tail.
		remaining = budget
		var result []Message
		for i := len(messages) - 1; i >= 0; i-- {
			t := estimateMessageTokens(messages[i])
			if remaining-t < 0 && len(result) > 0 {
				break
			}
			remaining -= t
			result = append([]Message{messages[i]}, result...)
		}
		return result
	}

	// Walk backwards from the end, accumulating recent messages that fit.
	var recentMessages []Message
	used := 0
	for i := len(messages) - 1; i >= 1; i-- {
		t := estimateMessageTokens(messages[i])
		if used+t > remaining {
			break
		}
		used += t
		recentMessages = append([]Message{messages[i]}, recentMessages...)
	}

	result := make([]Message, 0, 2+len(recentMessages))
	result = append(result, messages[0])
	result = append(result, Message{
		Role:    "user",
		Content: truncationPlaceholder,
	})
	result = append(result, recentMessages...)
	return result
}

func EstimateAllMessages(messages []Message) int {
	total := 0
	for _, m := range messages {
		total += estimateMessageTokens(m)
	}
	return total
}

func CopyMessages(messages []Message) []Message {
	out := make([]Message, len(messages))
	for i, m := range messages {
		cp := m
		if len(m.ToolUse) > 0 {
			cp.ToolUse = make([]ToolUseBlock, len(m.ToolUse))
			copy(cp.ToolUse, m.ToolUse)
		}
		if len(m.ToolResults) > 0 {
			cp.ToolResults = make([]ToolResultBlock, len(m.ToolResults))
			copy(cp.ToolResults, m.ToolResults)
		}
		out[i] = cp
	}
	return out
}

func collectMessages(messages []Message, drop []bool) []Message {
	var out []Message
	needsPlaceholder := false
	for i, m := range messages {
		if drop[i] {
			needsPlaceholder = true
			continue
		}
		if needsPlaceholder {
			// Insert a placeholder so the LLM knows there's a gap.
			out = append(out, Message{
				Role:    "user",
				Content: "[Some earlier assistant messages and tool results were removed to save context space. All your original messages are preserved.]",
			})
			needsPlaceholder = false
		}
		out = append(out, m)
	}
	return out
}
