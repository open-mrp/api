package router

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/augno/api/shared/appctx"

	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apierror "github.com/augno/api/shared/errors"
)

type router struct {
	*http.ServeMux
	routes      []Route
	handlers    map[string]map[string]http.HandlerFunc
	middlewares []func(http.HandlerFunc) http.HandlerFunc
}

func NewRouter() *router {
	router := &router{
		ServeMux:    http.NewServeMux(),
		routes:      make([]Route, 0),
		handlers:    make(map[string]map[string]http.HandlerFunc),
		middlewares: make([]func(http.HandlerFunc) http.HandlerFunc, 0),
	}

	router.ServeMux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/" {
			if rootHandlers, exists := router.handlers["/"]; exists {
				handler, methodExists := rootHandlers[req.Method]
				if !methodExists && req.Method == http.MethodOptions {
					for _, h := range rootHandlers {
						handler = h
						methodExists = true
						break
					}
				}

				if methodExists {
					ctx := appctx.WithRoutePattern(req.Context(), "/")
					ctx = appctx.WithAllowedMethods(ctx, collectMethods(rootHandlers))
					req = req.WithContext(ctx)

					finalHandler := handler
					for i := len(router.middlewares) - 1; i >= 0; i-- {
						finalHandler = router.middlewares[i](finalHandler)
					}
					finalHandler(w, req)
					return
				} else {
					httptransport.RespondWithAPIError(req.Context(), w, apierror.NewMethodNotAllowedError(fmt.Sprintf("Method %s not allowed for path %s.", req.Method, req.URL.Path)))
					return
				}
			}
		}

		var matchedRoute *Route
		var matchedParams map[string]string
		var allowedMethods []string

		for i, route := range router.routes {
			pathMatches := false
			var params map[string]string

			if route.PathPattern != nil {
				params = extractPathParams(route.PathPattern, route.PathParams, req.URL.Path)
				pathMatches = params != nil
			} else {
				pathMatches = route.Path == req.URL.Path
			}

			if pathMatches {
				allowedMethods = append(allowedMethods, route.Method)
				if route.Method == req.Method {
					matchedRoute = &router.routes[i]
					matchedParams = params
				} else if req.Method == http.MethodOptions && matchedRoute == nil {
					matchedRoute = &router.routes[i]
					matchedParams = params
				}
			}
		}

		if matchedRoute != nil {
			slices.Sort(allowedMethods)

			ctx := req.Context()
			if matchedParams != nil {
				ctx = appctx.WithPathParams(ctx, matchedParams)
			}
			ctx = appctx.WithRoutePattern(ctx, matchedRoute.Path)
			ctx = appctx.WithAllowedMethods(ctx, allowedMethods)
			req = req.WithContext(ctx)

			finalHandler := matchedRoute.Handler
			for i := len(router.middlewares) - 1; i >= 0; i-- {
				finalHandler = router.middlewares[i](finalHandler)
			}
			finalHandler(w, req)
		} else if len(allowedMethods) > 0 {
			httptransport.RespondWithAPIError(req.Context(), w, apierror.NewMethodNotAllowedError(fmt.Sprintf("Method %s not allowed for path %s.", req.Method, req.URL.Path)))
		} else {
			httptransport.RespondWithAPIError(req.Context(), w, apierror.NewResourceNotFoundError(fmt.Sprintf("The requested endpoint %s %s was not found.", req.Method, req.URL.Path)))
		}
	})

	return router
}

func (r *router) AddMiddleware(middleware func(http.HandlerFunc) http.HandlerFunc) {
	r.middlewares = append(r.middlewares, middleware)
}

func (r *router) HandleEndpoint(method, path string, handler http.HandlerFunc, isPublic bool) {
	r.handle(method, path, handler)
	r.routes[len(r.routes)-1].IsPublic = isPublic
}

func (r *router) handle(method, path string, handler http.HandlerFunc) {
	pattern, paramNames := compilePathPattern(path)

	route := Route{
		Method:      method,
		Path:        path,
		Handler:     handler,
		PathPattern: pattern,
		PathParams:  paramNames,
	}

	r.routes = append(r.routes, route)

	if pattern == nil {
		if r.handlers[path] == nil {
			r.handlers[path] = make(map[string]http.HandlerFunc)

			r.ServeMux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
				methodHandlers := r.handlers[path]

				handler, exists := methodHandlers[req.Method]
				if !exists && req.Method == http.MethodOptions {
					for _, h := range methodHandlers {
						handler = h
						exists = true
						break
					}
				}

				if exists {
					ctx := appctx.WithRoutePattern(req.Context(), path)
					ctx = appctx.WithAllowedMethods(ctx, collectMethods(methodHandlers))
					req = req.WithContext(ctx)

					finalHandler := handler
					for i := len(r.middlewares) - 1; i >= 0; i-- {
						finalHandler = r.middlewares[i](finalHandler)
					}
					finalHandler(w, req)
				} else {
					httptransport.RespondWithAPIError(req.Context(), w, apierror.NewMethodNotAllowedError(fmt.Sprintf("Method %s not allowed for path %s.", req.Method, req.URL.Path)))
				}
			})
		}

		r.handlers[path][method] = handler
	}
}

func (r *router) Get(path string, handler http.HandlerFunc) {
	r.handle("GET", path, handler)
}

func (r *router) Post(path string, handler http.HandlerFunc) {
	r.handle("POST", path, handler)
}

func (r *router) Put(path string, handler http.HandlerFunc) {
	r.handle("PUT", path, handler)
}

func (r *router) Patch(path string, handler http.HandlerFunc) {
	r.handle("PATCH", path, handler)
}

func (r *router) Delete(path string, handler http.HandlerFunc) {
	r.handle("DELETE", path, handler)
}

func (r *router) RegisterWithMiddleware(method, path string, handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) {
	finalHandler := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		finalHandler = middlewares[i](finalHandler)
	}

	for i := len(r.middlewares) - 1; i >= 0; i-- {
		finalHandler = r.middlewares[i](finalHandler)
	}

	r.handle(method, path, finalHandler)
}

func (r *router) GetRoutes() []any {
	routes := make([]any, len(r.routes))
	for i, route := range r.routes {
		routes[i] = map[string]any{
			"Method":      route.Method,
			"Path":        route.Path,
			"PathPattern": route.PathPattern,
			"Public":      route.IsPublic,
		}
	}
	return routes
}

func collectMethods(handlers map[string]http.HandlerFunc) []string {
	methods := make([]string, 0, len(handlers))
	for m := range handlers {
		methods = append(methods, m)
	}
	slices.Sort(methods)
	return methods
}
