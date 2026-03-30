package builtin

import (
	"bufio"
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

type nativeSSEClient struct {
	mu     sync.Mutex
	body   io.ReadCloser
	reader *bufio.Reader
	closed bool
}

type nativeSSEStream struct {
	mu      sync.Mutex
	writer  nethttp.ResponseWriter
	flusher nethttp.Flusher
	ctx     context.Context
	closed  bool
}

type nativeSSEHandler struct {
	fn  object.Object
	env *object.Environment
}

type nativeSSEServer struct {
	mu              sync.Mutex
	handlers        map[string]nativeSSEHandler
	server          *nethttp.Server
	listener        net.Listener
	addr            string
	startedAt       time.Time
	requestCount    int64
	connectionCount int64
	active          map[*nativeSSEStream]struct{}
}

func newSSELib() *object.Hash {
	return hashObject(map[string]object.Object{
		"client": newSSEClientLib(),
		"server": newSSEServerLib(),
	})
}

func newSSEClientLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"connect": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 && len(args) != 2 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1 or 2", len(args))
			}
			url, errObj := stringArg(args[0], "first argument to `connect` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}

			req, err := nethttp.NewRequest(nethttp.MethodGet, url, nil)
			if err != nil {
				return object.NewError(0, "could not build sse request: %s", err.Error())
			}
			req.Header.Set("Accept", "text/event-stream")

			if len(args) == 2 {
				headerHash, errObj := hashArg(args[1], "second argument to `connect` must be HASH, got %s")
				if errObj != nil {
					return errObj
				}
				applyHeaders(req.Header, headerHash)
			}

			resp, err := defaultHTTPClient.Do(req)
			if err != nil {
				return object.NewError(0, "could not connect sse '%s': %s", url, err.Error())
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				return object.NewError(0, "sse connect failed: status=%d body=%s", resp.StatusCode, string(body))
			}

			return newSSEClientObject(&nativeSSEClient{
				body:   resp.Body,
				reader: bufio.NewReader(resp.Body),
			})
		}),
	})
}

func newSSEServerLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"new": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return newSSEServerObject()
		}),
	})
}

func newSSEServerObject() *object.Hash {
	state := &nativeSSEServer{
		handlers: make(map[string]nativeSSEHandler),
		active:   make(map[*nativeSSEStream]struct{}),
	}

	return hashObject(map[string]object.Object{
		"handle": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 2 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=2", len(args))
			}
			path, errObj := stringArg(args[0], "first argument to `handle` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			switch args[1].(type) {
			case *object.Function, *object.Builtin:
			default:
				return object.NewError(0, "second argument to `handle` must be FUNCTION or BUILTIN, got %s", args[1].Type())
			}

			state.mu.Lock()
			state.handlers[path] = nativeSSEHandler{fn: args[1], env: env}
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
				return object.NewError(0, "sse server already running")
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
					handler, ok := state.handlers[r.URL.Path]
					state.mu.Unlock()
					if !ok {
						nethttp.NotFound(w, r)
						return
					}

					flusher, ok := w.(nethttp.Flusher)
					if !ok {
						nethttp.Error(w, "streaming unsupported", nethttp.StatusInternalServerError)
						return
					}

					w.Header().Set("Content-Type", "text/event-stream")
					w.Header().Set("Cache-Control", "no-cache")
					w.Header().Set("Connection", "keep-alive")
					w.WriteHeader(nethttp.StatusOK)
					flusher.Flush()

					stream := &nativeSSEStream{
						writer:  w,
						flusher: flusher,
						ctx:     r.Context(),
					}

					state.mu.Lock()
					state.connectionCount++
					state.active[stream] = struct{}{}
					state.mu.Unlock()

					defer func() {
						state.mu.Lock()
						delete(state.active, stream)
						state.mu.Unlock()
						stream.close()
					}()

					reqObject, errObj := buildHTTPRequestObject(r)
					if errObj != nil {
						fmt.Fprintln(os.Stderr, errObj.Inspect())
						return
					}

					result := handler.env.Call(handler.fn, []object.Object{reqObject, newSSEStreamObject(stream)}, 0)
					if errObj, ok := result.(*object.Error); ok {
						fmt.Fprintln(os.Stderr, errObj.Inspect())
						return
					}

					if errObj := writeSSEHandlerResult(stream, result); errObj != nil {
						fmt.Fprintln(os.Stderr, errObj.Inspect())
					}
				}),
			}

			state.server = server
			state.listener = listener
			state.addr = normalizeServerAddr(listener.Addr().String())
			state.startedAt = time.Now()
			state.requestCount = 0
			state.connectionCount = 0
			state.mu.Unlock()

			env.Go(func() {
				if err := server.Serve(listener); err != nil && err != nethttp.ErrServerClosed {
					fmt.Fprintln(os.Stderr, "ERROR: sse server serve failed:", err)
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
			active := make([]*nativeSSEStream, 0, len(state.active))
			for stream := range state.active {
				active = append(active, stream)
			}
			state.server = nil
			state.listener = nil
			state.mu.Unlock()

			for _, stream := range active {
				stream.close()
			}

			if server == nil {
				return &object.Null{}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := server.Shutdown(ctx); err != nil {
				return object.NewError(0, "could not stop sse server: %s", err.Error())
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
				"addr":             &object.String{Value: state.addr},
				"is_running":       boolObject(state.server != nil),
				"handler_count":    &object.Integer{Value: int64(len(state.handlers))},
				"request_count":    &object.Integer{Value: state.requestCount},
				"connection_count": &object.Integer{Value: state.connectionCount},
				"active_count":     &object.Integer{Value: int64(len(state.active))},
				"uptime_ms":        &object.Integer{Value: uptimeMs},
			})
		}),
	})
}

func newSSEClientObject(state *nativeSSEClient) *object.Hash {
	return hashObject(map[string]object.Object{
		"read": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return state.readEvent()
		}),
		"close": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			state.close()
			return &object.Null{}
		}),
		"is_closed": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return boolObject(state.isClosed())
		}),
	})
}

func newSSEStreamObject(state *nativeSSEStream) *object.Hash {
	return hashObject(map[string]object.Object{
		"send": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			data, errObj := stringArg(args[0], "argument to `send` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			return state.send(nativeSSEEvent{data: data})
		}),
		"send_event": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 2 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=2", len(args))
			}
			eventName, errObj := stringArg(args[0], "first argument to `send_event` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			data, errObj := stringArg(args[1], "second argument to `send_event` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			return state.send(nativeSSEEvent{event: eventName, data: data})
		}),
		"send_with_id": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 2 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=2", len(args))
			}
			data, errObj := stringArg(args[0], "first argument to `send_with_id` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			id, errObj := stringArg(args[1], "second argument to `send_with_id` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			return state.send(nativeSSEEvent{data: data, id: id})
		}),
		"send_event_with_id": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 3 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=3", len(args))
			}
			eventName, errObj := stringArg(args[0], "first argument to `send_event_with_id` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			data, errObj := stringArg(args[1], "second argument to `send_event_with_id` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			id, errObj := stringArg(args[2], "third argument to `send_event_with_id` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			return state.send(nativeSSEEvent{event: eventName, data: data, id: id})
		}),
		"set_retry": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			retryMs, errObj := integerArg(args[0], "first argument to `set_retry` must be INTEGER, got %s")
			if errObj != nil {
				return errObj
			}
			return state.send(nativeSSEEvent{retry: retryMs})
		}),
		"send_json": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			payload, err := json.Marshal(nativeValue(args[0]))
			if err != nil {
				return object.NewError(0, "could not encode sse json: %s", err.Error())
			}
			return state.send(nativeSSEEvent{event: "json", data: string(payload)})
		}),
		"close": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			state.close()
			return &object.Null{}
		}),
		"is_closed": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return boolObject(state.isClosed())
		}),
	})
}

func (c *nativeSSEClient) readEvent() object.Object {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.reader == nil {
		return &object.Null{}
	}

	eventType := "message"
	eventID := ""
	retry := ""
	dataLines := make([]string, 0, 1)

	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				c.closeLocked()
				if len(dataLines) == 0 && eventID == "" && retry == "" && eventType == "message" {
					return &object.Null{}
				}
				break
			}
			c.closeLocked()
			return object.NewError(0, "could not read sse event: %s", err.Error())
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(line, ":") {
			continue
		}

		field := line
		value := ""
		if idx := strings.Index(line, ":"); idx >= 0 {
			field = line[:idx]
			value = strings.TrimPrefix(line[idx+1:], " ")
		}

		switch field {
		case "event":
			eventType = value
		case "data":
			dataLines = append(dataLines, value)
		case "id":
			eventID = value
		case "retry":
			retry = value
		}
	}

	return hashObject(map[string]object.Object{
		"event": &object.String{Value: eventType},
		"data":  &object.String{Value: strings.Join(dataLines, "\n")},
		"id":    &object.String{Value: eventID},
		"retry": &object.String{Value: retry},
	})
}

func (c *nativeSSEClient) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeLocked()
}

func (c *nativeSSEClient) closeLocked() {
	if c.closed {
		return
	}
	if c.body != nil {
		_ = c.body.Close()
	}
	c.body = nil
	c.reader = nil
	c.closed = true
}

func (c *nativeSSEClient) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

type nativeSSEEvent struct {
	event string
	data  string
	id    string
	retry int64
}

func (s *nativeSSEStream) send(event nativeSSEEvent) object.Object {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.writer == nil {
		return object.NewError(0, "sse stream is closed")
	}
	select {
	case <-s.ctx.Done():
		s.closed = true
		return object.NewError(0, "sse stream is closed")
	default:
	}

	var builder strings.Builder
	if event.event != "" {
		builder.WriteString("event: ")
		builder.WriteString(event.event)
		builder.WriteString("\n")
	}
	if event.id != "" {
		builder.WriteString("id: ")
		builder.WriteString(event.id)
		builder.WriteString("\n")
	}
	if event.retry > 0 {
		builder.WriteString("retry: ")
		builder.WriteString(fmt.Sprintf("%d", event.retry))
		builder.WriteString("\n")
	}
	if event.data != "" {
		for _, line := range strings.Split(event.data, "\n") {
			builder.WriteString("data: ")
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\n")

	if _, err := io.WriteString(s.writer, builder.String()); err != nil {
		s.closed = true
		return object.NewError(0, "could not write sse event: %s", err.Error())
	}
	s.flusher.Flush()
	return &object.Null{}
}

func (s *nativeSSEStream) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	s.writer = nil
	s.flusher = nil
}

func (s *nativeSSEStream) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return true
	}
	select {
	case <-s.ctx.Done():
		return true
	default:
		return false
	}
}

func writeSSEHandlerResult(stream *nativeSSEStream, result object.Object) *object.Error {
	switch value := result.(type) {
	case nil, *object.Null:
		return nil
	case *object.String:
		if errObj, ok := stream.send(nativeSSEEvent{data: value.Value}).(*object.Error); ok {
			return errObj
		}
		return nil
	default:
		payload, err := json.Marshal(nativeValue(result))
		if err != nil {
			return object.NewError(0, "could not encode sse handler result: %s", err.Error())
		}
		if errObj, ok := stream.send(nativeSSEEvent{event: "json", data: string(payload)}).(*object.Error); ok {
			return errObj
		}
		return nil
	}
}
