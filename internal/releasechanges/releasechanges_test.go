package releasechanges

import (
	"reflect"
	"testing"
)

func TestAnalyze_ServiceLocalGoChange(t *testing.T) {
	t.Parallel()

	analysis := Analyze([]string{
		"services/core-service/internal/service/customer_service.go",
	}, map[string][]string{
		"services/core-service/internal/service": {"core-service"},
	})

	assertServicesEqual(t, analysis.Services, []string{"core-service"})
}

func TestAnalyze_ImportablePackageMarksDependentServices(t *testing.T) {
	t.Parallel()

	analysis := Analyze([]string{
		"services/auth-service/pkg/types/permissions.go",
	}, map[string][]string{
		"services/auth-service/pkg/types": {"api-gateway", "auth-service", "core-service"},
	})

	assertServicesEqual(t, analysis.Services, []string{"api-gateway", "auth-service", "core-service"})
}

// Cluster state moved to the private augno/infra repo, which reconciles its own manifests. A path
// that looks like one is now just an unrecognised file and must not select anything to rebuild.
func TestAnalyze_ManifestPathSelectsNothing(t *testing.T) {
	t.Parallel()

	analysis := Analyze([]string{
		"infra/production/kubernetes/apps/auth-service.yaml",
		"infra/production/kubernetes/config/app-config.yaml",
		"infra/production/terraform/eks.tf",
	}, nil)

	assertServicesEqual(t, analysis.Services, nil)
}

// The production Dockerfile stays in this repo: it is the build recipe for every service image.
func TestAnalyze_DockerfileChangeBuildsAllServices(t *testing.T) {
	t.Parallel()

	analysis := Analyze([]string{
		"infra/production/docker/Dockerfile",
	}, nil)

	assertServicesEqual(t, analysis.Services, ServiceNames)
}

func TestAnalyze_UnmappedSharedChangeBuildsAllServices(t *testing.T) {
	t.Parallel()

	analysis := Analyze([]string{
		"shared/db/migrations/0001_initial.sql",
	}, nil)

	assertServicesEqual(t, analysis.Services, ServiceNames)
}

func TestAnalyze_DocsOnlyChangeDoesNothing(t *testing.T) {
	t.Parallel()

	analysis := Analyze([]string{
		"docs/patterns/architecture-patterns.md",
	}, nil)

	assertServicesEqual(t, analysis.Services, nil)
}

func TestAnalyze_ProtoChangeBuildsAllServices(t *testing.T) {
	t.Parallel()

	analysis := Analyze([]string{
		"proto/core/core.proto",
	}, nil)

	assertServicesEqual(t, analysis.Services, ServiceNames)
}

func assertServicesEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("services mismatch: got %v want %v", got, want)
	}
}
