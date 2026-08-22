package releasechanges

import (
	"path/filepath"
	"slices"
	"strings"
)

var ServiceNames = []string{
	"api-gateway",
	"auth-service",
	"core-service",
	"platform-service",
	"notification-service",
	"billing-service",
	"agent-service",
}

// Analysis names the services a release has to rebuild. Every rebuilt service is also redeployed, so
// one list serves both. Cluster state — Terraform, ConfigMaps, platform components, per-service
// manifests — lives in the private open-mrp/infra repo and is reconciled by that repo's own workflows,
// so nothing here classifies it.
type Analysis struct {
	Services []string
}

// AnalyzeAll returns every service, bypassing change detection.
//
// Needed when a release has to reach the cluster even though the diff touches no
// service code — a fixed build pipeline being the usual case: the commit that
// repairs it changes only workflow files, so ordinary detection selects nothing
// and the previously failed images never get rebuilt.
func AnalyzeAll() Analysis {
	buildSet := make(map[string]struct{}, len(ServiceNames))
	addAllServices(buildSet)
	return Analysis{Services: orderedServices(buildSet)}
}

func Analyze(changedFiles []string, dirToServices map[string][]string) Analysis {
	buildSet := make(map[string]struct{}, len(ServiceNames))

	for _, rawPath := range changedFiles {
		path := normalizePath(rawPath)
		if path == "" {
			continue
		}

		if isGlobalBuildInput(path) {
			addAllServices(buildSet)
			continue
		}

		if dependentServices := dependentServicesForPath(path, dirToServices); len(dependentServices) > 0 {
			for _, service := range dependentServices {
				buildSet[service] = struct{}{}
			}
			continue
		}

		switch {
		case strings.HasPrefix(path, "shared/"), strings.HasPrefix(path, "proto/"):
			addAllServices(buildSet)
		default:
			if service, ok := serviceFromServicePath(path); ok {
				buildSet[service] = struct{}{}
			}
		}
	}

	return Analysis{Services: orderedServices(buildSet)}
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	cleaned := filepath.ToSlash(filepath.Clean(path))
	if cleaned == "." {
		return ""
	}

	return strings.TrimPrefix(cleaned, "./")
}

func isGlobalBuildInput(path string) bool {
	switch path {
	case ".dockerignore", "go.mod", "go.sum":
		return true
	}

	return strings.HasPrefix(path, "infra/production/docker/")
}

func dependentServicesForPath(path string, dirToServices map[string][]string) []string {
	dir := filepath.ToSlash(filepath.Dir(path))

	for dir != "." && dir != "/" && dir != "" {
		if services, ok := dirToServices[dir]; ok {
			return services
		}
		next := filepath.ToSlash(filepath.Dir(dir))
		if next == dir {
			break
		}
		dir = next
	}

	return nil
}

func serviceFromServicePath(path string) (string, bool) {
	if !strings.HasPrefix(path, "services/") {
		return "", false
	}

	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", false
	}

	service := parts[1]
	if !isKnownService(service) {
		return "", false
	}

	return service, true
}

func isKnownService(service string) bool {
	return slices.Contains(ServiceNames, service)
}

func addAllServices(target map[string]struct{}) {
	for _, service := range ServiceNames {
		target[service] = struct{}{}
	}
}

func orderedServices(source map[string]struct{}) []string {
	ordered := make([]string, 0, len(source))
	for _, service := range ServiceNames {
		if _, ok := source[service]; ok {
			ordered = append(ordered, service)
		}
	}
	return ordered
}
