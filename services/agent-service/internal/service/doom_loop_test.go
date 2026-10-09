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

	if d.Record("search", input) {
		t.Error("unexpected doom loop on call 1")
	}
	if d.Record("search", input) {
		t.Error("unexpected doom loop on call 2")
	}
	if !d.Record("search", input) {
		t.Error("expected doom loop to be detected on call 3")
	}
}

// Repeats count across the window, so interleaving another call does not hide a loop.
func TestDoomLoopDetector_CountsNonConsecutiveRepeats(t *testing.T) {
	t.Parallel()
	d := &doomLoopDetector{}
	input := json.RawMessage(`{"query":"test"}`)

	d.Record("search", input)
	d.Record("lookup", json.RawMessage(`{}`))
	d.Record("search", input)
	d.Record("lookup", json.RawMessage(`{"x":1}`))
	if !d.Record("search", input) {
		t.Error("third identical call in the window should be detected even when not consecutive")
	}
}

// Key order and whitespace differences are the same call.
func TestDoomLoopDetector_CanonicalizesJSON(t *testing.T) {
	t.Parallel()
	d := &doomLoopDetector{}
	d.Record("list_customers", json.RawMessage(`{"limit":10,"q":"acme"}`))
	d.Record("list_customers", json.RawMessage(`{ "q": "acme", "limit": 10 }`))
	if !d.Record("list_customers", json.RawMessage(`{"q":"acme","limit":10}`)) {
		t.Error("reordered keys should fingerprint identically")
	}
}

// URL tools are keyed on the normalized URL, so .md, trailing slashes, and fragments don't dodge detection.
func TestDoomLoopDetector_NormalizesURLTools(t *testing.T) {
	t.Parallel()
	d := &doomLoopDetector{}
	d.Record("fetch_url", json.RawMessage(`{"url":"https://Docs.Example.com/guide.md"}`))
	d.Record("fetch_url", json.RawMessage(`{"url":"https://docs.example.com/guide/"}`))
	if !d.Record("fetch_url", json.RawMessage(`{"url":"https://docs.example.com/guide#intro"}`)) {
		t.Error("spellings of the same page should count as repeats")
	}

	other := &doomLoopDetector{}
	other.Record("read_doc", json.RawMessage(`{"url":"https://docs.example.com/a"}`))
	other.Record("read_doc", json.RawMessage(`{"url":"https://docs.example.com/b"}`))
	if other.Record("read_doc", json.RawMessage(`{"url":"https://docs.example.com/c"}`)) {
		t.Error("different pages must not be treated as repeats")
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
	for i := range 25 {
		input, _ := json.Marshal(map[string]int{"i": i})
		d.Record("tool", input)
	}
	if len(d.history) > doomLoopWindowSize {
		t.Errorf("expected history to be trimmed to %d, got %d", doomLoopWindowSize, len(d.history))
	}
}

// A run of no-progress iterations steers once, then ends the turn if it continues.
func TestDoomLoopDetector_NoProgressSteersThenWrapsUp(t *testing.T) {
	t.Parallel()
	d := &doomLoopDetector{}
	var verdicts []loopVerdict
	for range noProgressWrapUpAfter {
		verdicts = append(verdicts, d.EndIteration(false))
	}
	for i, v := range verdicts {
		var want loopVerdict
		switch i + 1 {
		case noProgressSteerAfter:
			want = loopSteer
		case noProgressWrapUpAfter:
			want = loopWrapUp
		default:
			want = loopContinue
		}
		if v != want {
			t.Errorf("iteration %d: verdict %v, want %v", i+1, v, want)
		}
	}
}

// Progress resets the run, and the steering message is only ever sent once.
func TestDoomLoopDetector_ProgressResetsAndSteersOnce(t *testing.T) {
	t.Parallel()
	d := &doomLoopDetector{}
	for range noProgressSteerAfter {
		d.EndIteration(false)
	}
	if v := d.EndIteration(true); v != loopContinue {
		t.Fatalf("progress should continue, got %v", v)
	}
	for i := range noProgressWrapUpAfter - 1 {
		if v := d.EndIteration(false); v != loopContinue {
			t.Errorf("iteration %d after reset: got %v, want continue (already steered once)", i+1, v)
		}
	}
	if v := d.EndIteration(false); v != loopWrapUp {
		t.Errorf("sustained no-progress should wrap up, got %v", v)
	}
}

func TestMadeProgress(t *testing.T) {
	t.Parallel()
	if madeProgress("search_api_tools") || madeProgress("describe_api_operation") {
		t.Error("discovery tools alone are not progress")
	}
	if !madeProgress("list_customers") {
		t.Error("a successful operation is progress")
	}
}
