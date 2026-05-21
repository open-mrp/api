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

	assertServicesEqual(t, analysis.BuildServices, []string{"core-service"})
	assertServicesEqual(t, analysis.DeployServices, []string{"core-service"})
	assertBools(t, analysis, false, false, false)
}

func TestAnalyze_ImportablePackageMarksDependentServices(t *testing.T) {
	t.Parallel()

	analysis := Analyze([]string{
		"services/auth-service/pkg/types/permissions.go",
	}, map[string][]string{
		"services/auth-service/pkg/types": {"api-gateway", "auth-service", "core-service"},
	})

	assertServicesEqual(t, analysis.BuildServices, []string{"api-gateway", "auth-service", "core-service"})
	assertServicesEqual(t, analysis.DeployServices, []string{"api-gateway", "auth-service", "core-service"})
	assertBools(t, analysis, false, false, false)
}

func TestAnalyze_ManifestOnlyChangeDeploysOnlyThatService(t *testing.T) {
	t.Parallel()

	analysis := Analyze([]string{
		"infra/production/kubernetes/apps/auth-service.yaml",
	}, nil)

	assertServicesEqual(t, analysis.BuildServices, nil)
	assertServicesEqual(t, analysis.DeployServices, []string{"auth-service"})
	assertBools(t, analysis, false, false, false)
}

func TestAnalyze_ConfigChangeRedeploysAllServices(t *testing.T) {
	t.Parallel()

	analysis := Analyze([]string{
		"infra/production/kubernetes/config/app-config.yaml",
	}, nil)

	assertServicesEqual(t, analysis.BuildServices, nil)
	assertServicesEqual(t, analysis.DeployServices, ServiceNames)
	assertBools(t, analysis, false, true, false)
}

func TestAnalyze_UnmappedSharedChangeBuildsAllServices(t *testing.T) {
	t.Parallel()

	analysis := Analyze([]string{
		"shared/db/migrations/0001_initial.sql",
	}, nil)

	assertServicesEqual(t, analysis.BuildServices, ServiceNames)
	assertServicesEqual(t, analysis.DeployServices, ServiceNames)
	assertBools(t, analysis, false, false, false)
}

func TestAnalyze_DocsOnlyChangeDoesNothing(t *testing.T) {
	t.Parallel()

	analysis := Analyze([]string{
		"docs/patterns/architecture-patterns.md",
	}, nil)

	assertServicesEqual(t, analysis.BuildServices, nil)
	assertServicesEqual(t, analysis.DeployServices, nil)
	assertBools(t, analysis, false, false, false)
}

func TestAnalyze_ProtoChangeBuildsAllServices(t *testing.T) {
	t.Parallel()

	analysis := Analyze([]string{
		"proto/core/core.proto",
	}, nil)

	assertServicesEqual(t, analysis.BuildServices, ServiceNames)
	assertServicesEqual(t, analysis.DeployServices, ServiceNames)
	assertBools(t, analysis, false, false, false)
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

func assertBools(t *testing.T, analysis Analysis, terraform, config, platform bool) {
	t.Helper()
	if analysis.TerraformChanged != terraform {
		t.Fatalf("TerraformChanged mismatch: got %t want %t", analysis.TerraformChanged, terraform)
	}
	if analysis.ConfigChanged != config {
		t.Fatalf("ConfigChanged mismatch: got %t want %t", analysis.ConfigChanged, config)
	}
	if analysis.PlatformChanged != platform {
		t.Fatalf("PlatformChanged mismatch: got %t want %t", analysis.PlatformChanged, platform)
	}
}
