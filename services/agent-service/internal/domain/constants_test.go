package domain

import "testing"

func TestRunStatusIsCancellable(t *testing.T) {
	t.Parallel()

	cancellable := []string{
		RunStatusPending,
		RunStatusRunning,
		RunStatusAwaitingInput,
		RunStatusAwaitingApproval,
	}
	for _, s := range cancellable {
		if !RunStatusIsCancellable(s) {
			t.Errorf("expected status %q to be cancellable", s)
		}
	}

	terminal := []string{
		RunStatusCompleted,
		RunStatusFailed,
		RunStatusCancelled,
		"timed_out",
		"",
	}
	for _, s := range terminal {
		if RunStatusIsCancellable(s) {
			t.Errorf("expected status %q to NOT be cancellable", s)
		}
	}
}
