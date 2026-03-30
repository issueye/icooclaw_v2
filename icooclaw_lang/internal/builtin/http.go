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

type nativeHTTPRoute struct {
	statusCode int
	body       string
	headers    nethttp.Header
}

type nativeHTTPServer struct {
	mu       sync.Mutex
	routes   map[string]nativeHTTPRoute
	server   *nethttp.Server
	listener net.Listener
	addr     string
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
			if len(args) != 2 && len(args) != 3 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=2 or 3", len(args))
			}
			path, errObj := stringArg(args[0], "first argument to `route` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			body, errObj := stringArg(args[1], "second argument to `route` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			contentType := "text/plain; charset=utf-8"
			if len(args) == 3 {
				contentType, errObj = stringArg(args[2], "third argument to `route` must be STRING, got %s")
				if errObj != nil {
					return errObj
				}
			}
			state.mu.Lock()
			state.routes[path] = nativeHTTPRoute{
				statusCode: 200,
				body:       body,
				headers:    nethttp.Header{"Content-Type": []string{contentType}},
			}
			state.mu.Unlock()
			return &object.Null{}
		}),
		"route_json": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 2 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=2", len(args))
			}
			path, errObj := stringArg(args[0], "first argument to `route_json` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			payload, err := json.Marshal(nativeValue(args[1]))
			if err != nil {
				return object.NewError(0, "could not encode json response: %s", err.Error())
			}
			state.mu.Lock()
			state.routes[path] = nativeHTTPRoute{
				statusCode: 200,
				body:       string(payload),
				headers:    nethttp.Header{"Content-Type": []string{"application/json"}},
			}
			state.mu.Unlock()
			return &object.Null{}
		}),
		"route_response": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 2 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=2", len(args))
			}
			path, errObj := stringArg(args[0], "first argument to `route_response` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			responseHash, errObj := hashArg(args[1], "second argument to `route_response` must be HASH, got %s")
			if errObj != nil {
				return errObj
			}
			route, errObj := parseRouteResponse(responseHash)
			if errObj != nil {
				return errObj
			}
			state.mu.Lock()
			state.routes[path] = route
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
					route, ok := state.routes[r.URL.Path]
					state.mu.Unlock()
					if !ok {
						nethttp.NotFound(w, r)
						return
					}
					for key, values := range route.headers {
						for _, value := range values {
							w.Header().Add(key, value)
						}
					}
					w.WriteHeader(route.statusCode)
					_, _ = w.Write([]byte(route.body))
				}),
			}

			state.server = server
			state.listener = listener
			state.addr = normalizeServerAddr(listener.Addr().String())
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
		statusCode: 200,
		body:       "",
		headers:    make(nethttp.Header),
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
	if pair, ok := hash.Pairs["headers"]; ok {
		headerHash, errObj := hashArg(pair.Value, "field `headers` must be HASH, got %s")
		if errObj != nil {
			return route, errObj
		}
		applyHeaders(route.headers, headerHash)
	}
	return route, nil
}

func nativeValue(obj object.Object) interface{} {
	switch value := obj.(type) {
	case *object.String:
		return value.Value
	case *object.Integer:
		return value.Value
	case *object.Float:
		return value.Value
	case *object.Boolean:
		return value.Value
	case *object.Null:
		return nil
	case *object.Array:
		result := make([]interface{}, 0, len(value.Elements))
		for _, item := range value.Elements {
			result = append(result, nativeValue(item))
		}
		return result
	case *object.Hash:
		result := make(map[string]interface{}, len(value.Pairs))
		for _, pair := range value.Pairs {
			result[pair.Key.Inspect()] = nativeValue(pair.Value)
		}
		return result
	default:
		return value.Inspect()
	}
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
