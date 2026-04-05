package service

import (
	"encoding/json"
	"testing"
)

func TestDoomLoopDetector_NoLoopOnDifferentInputs(t *testing.T) {
	t.Parallel()
	d := &doomLoopDetector{}
	for i := range 5 {
		input, _ := json.Marshal(map[string]int{"page": i})
		if d.Record("search", input) {
			t.Errorf("unexpected doom loop on iteration %d with different inputs", i)
		}
	}
}

func TestDoomLoopDetector_DetectsIdenticalCalls(t *testing.T) {
	t.Parallel()
	d := &doomLoopDetector{}
	input := json.RawMessage(`{"query":"test"}`)

	// First two calls should not trigger.
	if d.Record("search", input) {
		t.Error("unexpected doom loop on call 1")
	}
	if d.Record("search", input) {
		t.Error("unexpected doom loop on call 2")
	}

	// Third identical call should trigger.
	if !d.Record("search", input) {
		t.Error("expected doom loop to be detected on call 3")
	}
}

func TestDoomLoopDetector_ResetsOnDifferentCall(t *testing.T) {
	t.Parallel()
	d := &doomLoopDetector{}
	input := json.RawMessage(`{"query":"test"}`)

	d.Record("search", input)
	d.Record("search", input)
	// Break the streak with a different tool.
	d.Record("lookup", json.RawMessage(`{}`))
	// Restart — should not trigger yet.
	if d.Record("search", input) {
		t.Error("unexpected doom loop after streak was broken")
	}
}

func TestDoomLoopDetector_DifferentToolsSameInput(t *testing.T) {
	t.Parallel()
	d := &doomLoopDetector{}
	input := json.RawMessage(`{"query":"test"}`)

	d.Record("search", input)
	d.Record("lookup", input)
	if d.Record("fetch", input) {
		t.Error("different tools with same input should not trigger doom loop")
	}
}

func TestDoomLoopDetector_WindowTrimming(t *testing.T) {
	t.Parallel()
	d := &doomLoopDetector{}
	// Fill history with diverse calls
	for i := range 25 {
		input, _ := json.Marshal(map[string]int{"i": i})
		d.Record("tool", input)
	}
	if len(d.history) > doomLoopWindowSize {
		t.Errorf("expected history to be trimmed to %d, got %d", doomLoopWindowSize, len(d.history))
	}
}
