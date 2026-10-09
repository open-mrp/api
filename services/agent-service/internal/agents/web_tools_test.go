package agents

import "testing"

func TestFetchStatusError(t *testing.T) {
	t.Parallel()
	for _, ok := range []int{200, 203, 299} {
		if err := fetchStatusError(ok); err != nil {
			t.Errorf("status %d should succeed, got %v", ok, err)
		}
	}
	for _, bad := range []int{301, 404, 410, 500} {
		if err := fetchStatusError(bad); err == nil {
			t.Errorf("status %d should be an error so the tool result is is_error", bad)
		}
	}
}
