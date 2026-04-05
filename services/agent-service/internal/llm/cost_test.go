package llm

import "testing"

func TestEstimateTokenCostCents(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		inputTokens  int
		outputTokens int
		model        string
		wantCents    int64
	}{
		{
			name:         "claude-sonnet-4 small usage",
			inputTokens:  1_000_000,
			outputTokens: 100_000,
			model:        "claude-sonnet-4",
			wantCents:    300 + 150, // 300 input + 150 output
		},
		{
			name:         "claude-haiku-4.5",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			model:        "claude-haiku-4.5",
			wantCents:    80 + 400,
		},
		{
			name:         "gpt-4o",
			inputTokens:  2_000_000,
			outputTokens: 500_000,
			model:        "gpt-4o",
			wantCents:    500 + 500,
		},
		{
			name:         "gpt-4o-mini",
			inputTokens:  10_000_000,
			outputTokens: 1_000_000,
			model:        "gpt-4o-mini",
			wantCents:    150 + 60,
		},
		{
			name:         "unknown model uses default (most expensive)",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			model:        "unknown-model",
			wantCents:    300 + 1500,
		},
		{
			name:         "zero tokens",
			inputTokens:  0,
			outputTokens: 0,
			model:        "claude-sonnet-4",
			wantCents:    0,
		},
		{
			name:         "sub-million tokens truncate via integer division",
			inputTokens:  999,
			outputTokens: 999,
			model:        "claude-sonnet-4",
			wantCents:    1, // 999*1500/1_000_000 = 1 (output), 999*300/1_000_000 = 0 (input)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokenCostCents(tt.inputTokens, tt.outputTokens, tt.model)
			if got != tt.wantCents {
				t.Errorf("EstimateTokenCostCents(%d, %d, %q) = %d, want %d",
					tt.inputTokens, tt.outputTokens, tt.model, got, tt.wantCents)
			}
		})
	}
}
