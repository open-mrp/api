package service

import (
	"crypto/sha256"
	"encoding/hex"
)

const (
	// doomLoopThreshold is the number of consecutive identical tool calls before injecting a synthetic error to break the loop.
	doomLoopThreshold = 3

	// doomLoopWindowSize is the maximum number of recent tool calls tracked.
	doomLoopWindowSize = 20
)

// toolCallFingerprint is a (toolName, inputHash) tuple identifying a unique tool invocation.
type toolCallFingerprint struct {
	Name      string
	InputHash string
}

// doomLoopDetector tracks recent tool calls and detects when an agent is stuck calling the same tool with identical input repeatedly.
type doomLoopDetector struct {
	history []toolCallFingerprint
}

// Record adds a tool call to the history and returns true if the same (name, inputHash) has been called doomLoopThreshold consecutive times.
func (d *doomLoopDetector) Record(toolName string, input []byte) bool {
	h := sha256.Sum256(input)
	fp := toolCallFingerprint{
		Name:      toolName,
		InputHash: hex.EncodeToString(h[:]),
	}

	d.history = append(d.history, fp)

	// Trim to window size.
	if len(d.history) > doomLoopWindowSize {
		d.history = d.history[len(d.history)-doomLoopWindowSize:]
	}

	// Check if the last N entries are identical.
	if len(d.history) < doomLoopThreshold {
		return false
	}

	for i := len(d.history) - doomLoopThreshold; i < len(d.history); i++ {
		if d.history[i] != fp {
			return false
		}
	}

	return true
}
