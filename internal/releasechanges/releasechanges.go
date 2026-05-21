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

type Analysis struct {
	BuildServices    []string
	DeployServices   []string
	TerraformChanged bool
	ConfigChanged    bool
	PlatformChanged  bool
}

func Analyze(changedFiles []string, dirToServices map[string][]string) Analysis {
	buildSet := make(map[string]struct{}, len(ServiceNames))
	deploySet := make(map[string]struct{}, len(ServiceNames))

	analysis := Analysis{}

	for _, rawPath := range changedFiles {
		path := normalizePath(rawPath)
		if path == "" {
			continue
		}

		switch {
		case strings.HasPrefix(path, "infra/production/terraform/"):
			analysis.TerraformChanged = true
			continue
		case strings.HasPrefix(path, "infra/production/kubernetes/platform/"):
			analysis.PlatformChanged = true
			continue
		case strings.HasPrefix(path, "infra/production/kubernetes/config/"):
			analysis.ConfigChanged = true
			continue
		}

		if service, ok := serviceFromAppManifestPath(path); ok {
			deploySet[service] = struct{}{}
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

	for service := range buildSet {
		deploySet[service] = struct{}{}
	}

	if analysis.ConfigChanged {
		addAllServices(deploySet)
	}

	analysis.BuildServices = orderedServices(buildSet)
	analysis.DeployServices = orderedServices(deploySet)

	return analysis
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

func serviceFromAppManifestPath(path string) (string, bool) {
	if !strings.HasPrefix(path, "infra/production/kubernetes/apps/") {
		return "", false
	}

	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if !isKnownService(base) {
		return "", false
	}

	return base, true
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
