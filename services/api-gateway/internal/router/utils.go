package router

import (
	"regexp"
)

func compilePathPattern(path string) (*regexp.Regexp, []string) {
	paramPattern := regexp.MustCompile(`\{([^}]+)\}`)
	var paramNames []string

	matches := paramPattern.FindAllStringSubmatch(path, -1)
	for _, match := range matches {
		paramNames = append(paramNames, match[1])
	}

	pattern := paramPattern.ReplaceAllString(path, `([^/]+)`)

	pattern = "^" + pattern + "$"

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, paramNames
	}

	return regex, paramNames
}

func extractPathParams(pattern *regexp.Regexp, paramNames []string, path string) map[string]string {
	if pattern == nil {
		return nil
	}

	matches := pattern.FindStringSubmatch(path)
	if len(matches) != len(paramNames)+1 {
		return nil
	}

	params := make(map[string]string)
	for i, name := range paramNames {
		params[name] = matches[i+1]
	}

	return params
}
