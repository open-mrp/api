package buildinfo

import "testing"

func TestSourceRefPrefersRelease(t *testing.T) {
	prev := Release
	t.Cleanup(func() { Release = prev })

	Release = "v1.2.3"
	if got := SourceRef(); got != "v1.2.3" {
		t.Errorf("SourceRef() = %q, want the stamped release", got)
	}
}
