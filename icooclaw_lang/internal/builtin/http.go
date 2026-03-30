package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	nethttp "net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/issueye/icooclaw_lang/internal/object"
)

var defaultHTTPClient = &nethttp.Client{Timeout: 10 * time.Second}

const anyMethod = "*"

type nativeHTTPRoute struct {
	method     string
	statusCode int
	body       string
	headers    nethttp.Header
	filePath   string
	handler    object.Object
	handlerEnv *object.Environment
}

type nativeHTTPServer struct {
	mu           sync.Mutex
	routes       map[string]nativeHTTPRoute
	notFound     *nativeHTTPRoute
	server       *nethttp.Server
	listener     net.Listener
	addr         string
	startedAt    time.Time
	requestCount int64
}

func newHTTPLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"client": newHTTPClientLib(),
		"server": newHTTPServerLib(),
	})
}

func newHTTPClientLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"get": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 && len(args) != 2 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1 or 2", len(args))
			}
			url, errObj := stringArg(args[0], "first argument to `get` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			var headers *object.Hash
			if len(args) == 2 {
				headers, errObj = hashArg(args[1], "second argument to `get` must be HASH, got %s")
				if errObj != nil {
					return errObj
				}
			}
			return doHTTPRequest("GET", url, "", headers)
		}),
		"post": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 2 && len(args) != 3 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=2 or 3", len(args))
			}
			url, errObj := stringArg(args[0], "first argument to `post` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			body, errObj := stringArg(args[1], "second argument to `post` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			var headers *object.Hash
			if len(args) == 3 {
				headers, errObj = hashArg(args[2], "third argument to `post` must be HASH, got %s")
				if errObj != nil {
					return errObj
				}
			}
			return doHTTPRequest("POST", url, body, headers)
		}),
		"request": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 4 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=2..4", len(args))
			}
			method, errObj := stringArg(args[0], "first argument to `request` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			url, errObj := stringArg(args[1], "second argument to `request` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}

			body := ""
			var headers *object.Hash
			if len(args) >= 3 {
				if _, ok := args[2].(*object.Null); !ok {
					body, errObj = stringArg(args[2], "third argument to `request` must be STRING or NULL, got %s")
					if errObj != nil {
						return errObj
					}
				}
			}
			if len(args) == 4 {
				headers, errObj = hashArg(args[3], "fourth argument to `request` must be HASH, got %s")
				if errObj != nil {
					return errObj
				}
			}

			return doHTTPRequest(strings.ToUpper(method), url, body, headers)
		}),
	})
}

func newHTTPServerLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"new": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return newServerObject()
		}),
	})
}

func newServerObject() *object.Hash {
	state := &nativeHTTPServer{
		routes: make(map[string]nativeHTTPRoute),
	}

	return hashObject(map[string]object.Object{
		"route": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			method, path, payloadOffset, errObj := parseRouteTargetArgs(args, 2, 3)
			if errObj != nil {
				return errObj
			}

			body, errObj := stringArg(args[payloadOffset], "route body must be STRING, got %s")
			if errObj != nil {
				return errObj
			}

			contentType := "text/plain; charset=utf-8"
			if len(args) == payloadOffset+2 {
				contentType, errObj = stringArg(args[payloadOffset+1], "route content_type must be STRING, got %s")
				if errObj != nil {
					return errObj
				}
			}

			state.setRoute(path, nativeHTTPRoute{
				method:     method,
				statusCode: 200,
				body:       body,
				headers:    nethttp.Header{"Content-Type": []string{contentType}},
			})
			return &object.Null{}
		}),
		"route_json": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			method, path, payloadOffset, errObj := parseRouteTargetArgs(args, 2, 2)
			if errObj != nil {
				return errObj
			}

			payload, err := json.Marshal(nativeValue(args[payloadOffset]))
			if err != nil {
				return object.NewError(0, "could not encode json response: %s", err.Error())
			}

			state.setRoute(path, nativeHTTPRoute{
				method:     method,
				statusCode: 200,
				body:       string(payload),
				headers:    nethttp.Header{"Content-Type": []string{"application/json"}},
			})
			return &object.Null{}
		}),
		"route_response": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			method, path, payloadOffset, errObj := parseRouteTargetArgs(args, 2, 2)
			if errObj != nil {
				return errObj
			}

			responseHash, errObj := hashArg(args[payloadOffset], "route_response payload must be HASH, got %s")
			if errObj != nil {
				return errObj
			}

			route, errObj := parseRouteResponse(responseHash)
			if errObj != nil {
				return errObj
			}
			route.method = method
			state.setRoute(path, route)
			return &object.Null{}
		}),
		"route_file": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			method, path, payloadOffset, errObj := parseRouteTargetArgs(args, 2, 3)
			if errObj != nil {
				return errObj
			}

			filePath, errObj := stringArg(args[payloadOffset], "route_file file_path must be STRING, got %s")
			if errObj != nil {
				return errObj
			}

			headers := make(nethttp.Header)
			if len(args) == payloadOffset+2 {
				contentType, errObj := stringArg(args[payloadOffset+1], "route_file content_type must be STRING, got %s")
				if errObj != nil {
					return errObj
				}
				headers.Set("Content-Type", contentType)
			}

			state.setRoute(path, nativeHTTPRoute{
				method:     method,
				statusCode: 200,
				filePath:   filePath,
				headers:    headers,
			})
			return &object.Null{}
		}),
		"handle": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			method, path, payloadOffset, errObj := parseRouteTargetArgs(args, 2, 2)
			if errObj != nil {
				return errObj
			}

			switch args[payloadOffset].(type) {
			case *object.Function, *object.Builtin:
			default:
				return object.NewError(0, "handle handler must be FUNCTION or BUILTIN, got %s", args[payloadOffset].Type())
			}

			state.setRoute(path, nativeHTTPRoute{
				method:     method,
				statusCode: 200,
				headers:    make(nethttp.Header),
				handler:    args[payloadOffset],
				handlerEnv: env,
			})
			return &object.Null{}
		}),
		"not_found": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			responseHash, errObj := hashArg(args[0], "first argument to `not_found` must be HASH, got %s")
			if errObj != nil {
				return errObj
			}

			route, errObj := parseRouteResponse(responseHash)
			if errObj != nil {
				return errObj
			}

			state.mu.Lock()
			state.notFound = &route
			state.mu.Unlock()
			return &object.Null{}
		}),
		"start": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			addr, errObj := stringArg(args[0], "argument to `start` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}

			state.mu.Lock()
			if state.server != nil {
				state.mu.Unlock()
				return object.NewError(0, "http server already running")
			}

			listener, err := net.Listen("tcp4", addr)
			if err != nil {
				state.mu.Unlock()
				return object.NewError(0, "could not listen on '%s': %s", addr, err.Error())
			}

			server := &nethttp.Server{
				Handler: nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
					state.mu.Lock()
					state.requestCount++
					route, ok := state.lookupRoute(strings.ToUpper(r.Method), r.URL.Path)
					notFound := state.notFound
					state.mu.Unlock()

					if !ok {
						if notFound != nil {
							if err := writeRouteResponse(w, *notFound); err != nil {
								nethttp.Error(w, err.Error(), nethttp.StatusInternalServerError)
							}
							return
						}
						nethttp.NotFound(w, r)
						return
					}

					if route.handler != nil {
						responseRoute, errObj := invokeHTTPHandler(route, r)
						if errObj != nil {
							nethttp.Error(w, errObj.Message, nethttp.StatusInternalServerError)
							return
						}
						route = responseRoute
					}

					if err := writeRouteResponse(w, route); err != nil {
						nethttp.Error(w, err.Error(), nethttp.StatusInternalServerError)
					}
				}),
			}

			state.server = server
			state.listener = listener
			state.addr = normalizeServerAddr(listener.Addr().String())
			state.startedAt = time.Now()
			state.requestCount = 0
			state.mu.Unlock()

			env.Go(func() {
				if err := server.Serve(listener); err != nil && err != nethttp.ErrServerClosed {
					fmt.Fprintln(os.Stderr, "ERROR: http server serve failed:", err)
				}
			})

			return &object.String{Value: state.addr}
		}),
		"stop": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}

			state.mu.Lock()
			server := state.server
			state.server = nil
			state.listener = nil
			state.mu.Unlock()
			if server == nil {
				return &object.Null{}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := server.Shutdown(ctx); err != nil {
				return object.NewError(0, "could not stop http server: %s", err.Error())
			}
			return &object.Null{}
		}),
		"addr": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			state.mu.Lock()
			defer state.mu.Unlock()
			if state.addr == "" {
				return &object.Null{}
			}
			return &object.String{Value: state.addr}
		}),
		"url": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			path, errObj := stringArg(args[0], "first argument to `url` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			state.mu.Lock()
			defer state.mu.Unlock()
			if state.addr == "" {
				return &object.Null{}
			}
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			return &object.String{Value: "http://" + state.addr + path}
		}),
		"is_running": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			state.mu.Lock()
			defer state.mu.Unlock()
			return boolObject(state.server != nil)
		}),
		"stats": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			state.mu.Lock()
			defer state.mu.Unlock()

			uptimeMs := int64(0)
			if !state.startedAt.IsZero() {
				uptimeMs = time.Since(state.startedAt).Milliseconds()
			}

			return hashObject(map[string]object.Object{
				"addr":          &object.String{Value: state.addr},
				"is_running":    boolObject(state.server != nil),
				"route_count":   &object.Integer{Value: int64(len(state.routes))},
				"request_count": &object.Integer{Value: state.requestCount},
				"uptime_ms":     &object.Integer{Value: uptimeMs},
			})
		}),
	})
}

func doHTTPRequest(method, url, body string, headers *object.Hash) object.Object {
	req, err := nethttp.NewRequest(method, url, bytes.NewBufferString(body))
	if err != nil {
		return object.NewError(0, "could not build request: %s", err.Error())
	}

	if headers != nil {
		applyHeaders(req.Header, headers)
	}

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return object.NewError(0, "http request failed: %s", err.Error())
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return object.NewError(0, "could not read response body: %s", err.Error())
	}

	return responseObject(resp.Status, resp.StatusCode, string(data), resp.Header)
}

func responseObject(status string, statusCode int, body string, headers nethttp.Header) *object.Hash {
	headerPairs := make(map[string]object.Object, len(headers))
	for key, values := range headers {
		items := make([]object.Object, 0, len(values))
		for _, value := range values {
			items = append(items, &object.String{Value: value})
		}
		headerPairs[key] = &object.Array{Elements: items}
	}

	return hashObject(map[string]object.Object{
		"status":      &object.String{Value: status},
		"status_code": &object.Integer{Value: int64(statusCode)},
		"body":        &object.String{Value: body},
		"headers":     hashObject(headerPairs),
	})
}

func applyHeaders(dst nethttp.Header, headers *object.Hash) {
	for _, pair := range headers.Pairs {
		key := pair.Key.Inspect()
		switch value := pair.Value.(type) {
		case *object.String:
			dst.Set(key, value.Value)
		case *object.Array:
			for _, item := range value.Elements {
				dst.Add(key, item.Inspect())
			}
		default:
			dst.Set(key, value.Inspect())
		}
	}
}

func parseRouteResponse(hash *object.Hash) (nativeHTTPRoute, *object.Error) {
	route := nativeHTTPRoute{
		method:     anyMethod,
		statusCode: 200,
		body:       "",
		headers:    make(nethttp.Header),
	}

	if pair, ok := hash.Pairs["method"]; ok {
		method, errObj := stringArg(pair.Value, "field `method` must be STRING, got %s")
		if errObj != nil {
			return route, errObj
		}
		route.method = strings.ToUpper(method)
	}
	if pair, ok := hash.Pairs["status_code"]; ok {
		statusCode, errObj := integerArg(pair.Value, "field `status_code` must be INTEGER, got %s")
		if errObj != nil {
			return route, errObj
		}
		route.statusCode = int(statusCode)
	}
	if pair, ok := hash.Pairs["body"]; ok {
		body, errObj := stringArg(pair.Value, "field `body` must be STRING, got %s")
		if errObj != nil {
			return route, errObj
		}
		route.body = body
	}
	if pair, ok := hash.Pairs["file_path"]; ok {
		filePath, errObj := stringArg(pair.Value, "field `file_path` must be STRING, got %s")
		if errObj != nil {
			return route, errObj
		}
		route.filePath = filePath
	}
	if pair, ok := hash.Pairs["headers"]; ok {
		headerHash, errObj := hashArg(pair.Value, "field `headers` must be HASH, got %s")
		if errObj != nil {
			return route, errObj
		}
		applyHeaders(route.headers, headerHash)
	}
	return route, nil
}

func normalizeServerAddr(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}

func parseRouteTargetArgs(args []object.Object, legacyMinArgs, legacyMaxArgs int) (string, string, int, *object.Error) {
	if len(args) < legacyMinArgs || len(args) > legacyMaxArgs+1 {
		return "", "", 0, object.NewError(0, "wrong number of arguments. got=%d", len(args))
	}

	if len(args) >= legacyMinArgs+1 {
		if methodValue, ok := args[0].(*object.String); ok && looksLikeHTTPMethod(methodValue.Value) {
			path, errObj := stringArg(args[1], "route path must be STRING, got %s")
			if errObj != nil {
				return "", "", 0, errObj
			}
			return normalizeHTTPMethod(methodValue.Value), path, 2, nil
		}
	}

	path, errObj := stringArg(args[0], "route path must be STRING, got %s")
	if errObj != nil {
		return "", "", 0, errObj
	}
	return anyMethod, path, 1, nil
}

func looksLikeHTTPMethod(value string) bool {
	switch strings.ToUpper(value) {
	case anyMethod, "ANY", "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		return true
	default:
		return false
	}
}

func normalizeHTTPMethod(value string) string {
	value = strings.ToUpper(value)
	if value == "ANY" {
		return anyMethod
	}
	return value
}

func routeKey(method, path string) string {
	return method + " " + path
}

func (s *nativeHTTPServer) setRoute(path string, route nativeHTTPRoute) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if route.method == "" {
		route.method = anyMethod
	}
	s.routes[routeKey(normalizeHTTPMethod(route.method), path)] = route
}

func (s *nativeHTTPServer) lookupRoute(method, path string) (nativeHTTPRoute, bool) {
	if route, ok := s.routes[routeKey(normalizeHTTPMethod(method), path)]; ok {
		return route, true
	}
	route, ok := s.routes[routeKey(anyMethod, path)]
	return route, ok
}

func writeRouteResponse(w nethttp.ResponseWriter, route nativeHTTPRoute) error {
	for key, values := range route.headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	if route.filePath != "" {
		data, err := os.ReadFile(route.filePath)
		if err != nil {
			return err
		}
		if route.headers.Get("Content-Type") == "" {
			w.Header().Set("Content-Type", nethttp.DetectContentType(data))
		}
		w.WriteHeader(route.statusCode)
		_, err = w.Write(data)
		return err
	}

	w.WriteHeader(route.statusCode)
	_, err := w.Write([]byte(route.body))
	return err
}

func invokeHTTPHandler(route nativeHTTPRoute, r *nethttp.Request) (nativeHTTPRoute, *object.Error) {
	if route.handler == nil || route.handlerEnv == nil {
		return nativeHTTPRoute{}, object.NewError(0, "http handler is not configured")
	}

	requestObject, errObj := buildHTTPRequestObject(r)
	if errObj != nil {
		return nativeHTTPRoute{}, errObj
	}

	result := route.handlerEnv.Call(route.handler, []object.Object{requestObject}, 0)
	if errObj, ok := result.(*object.Error); ok {
		return nativeHTTPRoute{}, errObj
	}

	return responseRouteFromObject(result)
}

func buildHTTPRequestObject(r *nethttp.Request) (*object.Hash, *object.Error) {
	body := ""
	if r.Body != nil {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, object.NewError(0, "could not read request body: %s", err.Error())
		}
		body = string(data)
	}

	headerPairs := make(map[string]object.Object, len(r.Header))
	for key, values := range r.Header {
		items := make([]object.Object, 0, len(values))
		for _, value := range values {
			items = append(items, &object.String{Value: value})
		}
		headerPairs[key] = &object.Array{Elements: items}
	}

	queryPairs := make(map[string]object.Object, len(r.URL.Query()))
	for key, values := range r.URL.Query() {
		if len(values) == 1 {
			queryPairs[key] = &object.String{Value: values[0]}
			continue
		}
		items := make([]object.Object, 0, len(values))
		for _, value := range values {
			items = append(items, &object.String{Value: value})
		}
		queryPairs[key] = &object.Array{Elements: items}
	}

	return hashObject(map[string]object.Object{
		"method":    &object.String{Value: strings.ToUpper(r.Method)},
		"path":      &object.String{Value: r.URL.Path},
		"raw_query": &object.String{Value: r.URL.RawQuery},
		"query":     hashObject(queryPairs),
		"body":      &object.String{Value: body},
		"headers":   hashObject(headerPairs),
		"host":      &object.String{Value: r.Host},
	}), nil
}

func responseRouteFromObject(result object.Object) (nativeHTTPRoute, *object.Error) {
	switch value := result.(type) {
	case *object.Null:
		return nativeHTTPRoute{
			statusCode: 200,
			body:       "",
			headers:    nethttp.Header{"Content-Type": []string{"text/plain; charset=utf-8"}},
		}, nil
	case *object.String:
		return nativeHTTPRoute{
			statusCode: 200,
			body:       value.Value,
			headers:    nethttp.Header{"Content-Type": []string{"text/plain; charset=utf-8"}},
		}, nil
	case *object.Hash:
		if looksLikeHTTPResponseHash(value) {
			route, errObj := parseRouteResponse(value)
			if errObj != nil {
				return nativeHTTPRoute{}, errObj
			}
			if route.headers == nil {
				route.headers = make(nethttp.Header)
			}
			if route.filePath == "" && route.headers.Get("Content-Type") == "" {
				route.headers.Set("Content-Type", "text/plain; charset=utf-8")
			}
			return route, nil
		}
	}

	payload, err := json.Marshal(nativeValue(result))
	if err != nil {
		return nativeHTTPRoute{}, object.NewError(0, "could not encode handler response: %s", err.Error())
	}

	return nativeHTTPRoute{
		statusCode: 200,
		body:       string(payload),
		headers:    nethttp.Header{"Content-Type": []string{"application/json"}},
	}, nil
}

func looksLikeHTTPResponseHash(hash *object.Hash) bool {
	for _, key := range []string{"status_code", "body", "headers", "file_path", "method"} {
		if _, ok := hash.Pairs[key]; ok {
			return true
		}
	}
	return false
}
