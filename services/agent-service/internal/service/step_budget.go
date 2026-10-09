package service

import (
	"fmt"

	"github.com/open-mrp/api/shared/constants"
)

const (
	// defaultInteractiveMaxSteps is the per-turn LLM-call budget for chat and manual runs, where a person is waiting.
	defaultInteractiveMaxSteps = 30

	// defaultBackgroundMaxSteps is the per-turn LLM-call budget for scheduled and event runs.
	defaultBackgroundMaxSteps = 40

	// maxStepsCeiling is the highest max_steps an agent may configure.
	maxStepsCeiling = 60
)

// resolveMaxSteps returns the agent's configured step budget, or the default for how the run was triggered.
func resolveMaxSteps(configured int, triggerType string) int {
	if configured > 0 {
		return min(configured, maxStepsCeiling)
	}
	switch constants.AgentTriggerType(triggerType) {
	case constants.AgentTriggerTypeScheduled, constants.AgentTriggerTypeEvent:
		return defaultBackgroundMaxSteps
	default:
		return defaultInteractiveMaxSteps
	}
}

// stepsReminderAt is the 1-based step after which the agent is told how few steps remain.
func stepsReminderAt(maxSteps int) int {
	return maxSteps * 3 / 4
}

func stepsRemainingPrompt(remaining int) string {
	return fmt.Sprintf("You have %d steps left in this turn. Finish the task if you can. If you can't, answer now with what you have and say what is missing.", remaining)
}

// wrapUpReason says why the loop stopped before the agent finished on its own.
type wrapUpReason int

const (
	wrapUpMaxSteps wrapUpReason = iota
	wrapUpNoProgress
)

// wrapUpPrompt asks for a text-only final answer once tools are disabled.
func wrapUpPrompt(reason wrapUpReason) string {
	why := "You have reached the maximum number of steps for this turn"
	if reason == wrapUpNoProgress {
		why = "You have stopped making progress — repeated searches or errors"
	}
	return why + `, so tools are now disabled. Respond with text only and do not attempt any tool calls.

Write your final response to the user now. Include:
- What you accomplished.
- What remains unfinished.
- What is missing or blocked you, such as an operation, data, or permission your tools don't provide, and what the user could do next.`
}

// wrapUpFallback is the run output when the wrap-up call itself fails.
const wrapUpFallback = "I ran out of steps for this request before I could finish, and couldn't summarize my progress."
