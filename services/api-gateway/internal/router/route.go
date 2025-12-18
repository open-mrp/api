package router

import (
	"net/http"
	"regexp"
)

type Route struct {
	Method      string
	Path        string
	Handler     http.HandlerFunc
	PathPattern *regexp.Regexp
	PathParams  []string
}
