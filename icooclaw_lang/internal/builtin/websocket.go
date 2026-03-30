package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	nethttp "net/http"
	"os"
	"strings"
	"sync"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/issueye/icooclaw_lang/internal/object"
)

type nativeWebSocketConn struct {
	mu     sync.Mutex
	conn   *gws.Conn
	closed bool
}

type nativeWebSocketHandler struct {
	fn  object.Object
	env *object.Environment
}

type nativeWebSocketServer struct {
	mu              sync.Mutex
	handlers        map[string]nativeWebSocketHandler
	server          *nethttp.Server
	listener        net.Listener
	addr            string
	startedAt       time.Time
	requestCount    int64
	connectionCount int64
	active          map[*nativeWebSocketConn]struct{}
}

var websocketUpgrader = gws.Upgrader{
	CheckOrigin: func(r *nethttp.Request) bool { return true },
}

func newWebSocketLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"client": newWebSocketClientLib(),
		"server": newWebSocketServerLib(),
	})
}

func newWebSocketClientLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"connect": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 && len(args) != 2 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1 or 2", len(args))
			}
			url, errObj := stringArg(args[0], "first argument to `connect` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}

			headers := nethttp.Header{}
			if len(args) == 2 {
				headerHash, errObj := hashArg(args[1], "second argument to `connect` must be HASH, got %s")
				if errObj != nil {
					return errObj
				}
				applyHeaders(headers, headerHash)
			}

			conn, _, err := gws.DefaultDialer.Dial(url, headers)
			if err != nil {
				return object.NewError(0, "could not connect websocket '%s': %s", url, err.Error())
			}
			return newWebSocketConnObject(&nativeWebSocketConn{conn: conn})
		}),
	})
}

func newWebSocketServerLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"new": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return newWebSocketServerObject()
		}),
	})
}

func newWebSocketServerObject() *object.Hash {
	state := &nativeWebSocketServer{
		handlers: make(map[string]nativeWebSocketHandler),
		active:   make(map[*nativeWebSocketConn]struct{}),
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
			state.handlers[path] = nativeWebSocketHandler{fn: args[1], env: env}
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
				return object.NewError(0, "websocket server already running")
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

					conn, err := websocketUpgrader.Upgrade(w, r, nil)
					if err != nil {
						fmt.Fprintln(os.Stderr, "ERROR: websocket upgrade failed:", err)
						return
					}

					connState := &nativeWebSocketConn{conn: conn}
					state.mu.Lock()
					state.connectionCount++
					state.active[connState] = struct{}{}
					state.mu.Unlock()

					defer func() {
						state.mu.Lock()
						delete(state.active, connState)
						state.mu.Unlock()
						connState.close()
					}()

					reqObject, errObj := buildHTTPRequestObject(r)
					if errObj != nil {
						fmt.Fprintln(os.Stderr, errObj.Inspect())
						return
					}

					result := handler.env.Call(handler.fn, []object.Object{reqObject, newWebSocketConnObject(connState)}, 0)
					if errObj, ok := result.(*object.Error); ok {
						fmt.Fprintln(os.Stderr, errObj.Inspect())
						return
					}

					if errObj := writeWebSocketHandlerResult(connState, result); errObj != nil {
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
					fmt.Fprintln(os.Stderr, "ERROR: websocket server serve failed:", err)
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
			active := make([]*nativeWebSocketConn, 0, len(state.active))
			for conn := range state.active {
				active = append(active, conn)
			}
			state.server = nil
			state.listener = nil
			state.mu.Unlock()

			for _, conn := range active {
				conn.close()
			}

			if server == nil {
				return &object.Null{}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := server.Shutdown(ctx); err != nil {
				return object.NewError(0, "could not stop websocket server: %s", err.Error())
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
			return &object.String{Value: "ws://" + state.addr + path}
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

func newWebSocketConnObject(state *nativeWebSocketConn) *object.Hash {
	return hashObject(map[string]object.Object{
		"send": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			text, errObj := stringArg(args[0], "argument to `send` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}
			return state.writeMessage(gws.TextMessage, []byte(text))
		}),
		"send_json": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			payload, err := json.Marshal(nativeValue(args[0]))
			if err != nil {
				return object.NewError(0, "could not encode websocket json: %s", err.Error())
			}
			return state.writeMessage(gws.TextMessage, payload)
		}),
		"read": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			messageType, data, errObj := state.readMessage()
			if errObj != nil {
				return errObj
			}
			if messageType == 0 {
				return &object.Null{}
			}
			return &object.String{Value: string(data)}
		}),
		"read_message": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			messageType, data, errObj := state.readMessage()
			if errObj != nil {
				return errObj
			}
			if messageType == 0 {
				return &object.Null{}
			}
			return hashObject(map[string]object.Object{
				"type": &object.String{Value: websocketMessageTypeName(messageType)},
				"data": &object.String{Value: string(data)},
			})
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

func (c *nativeWebSocketConn) writeMessage(messageType int, payload []byte) object.Object {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.conn == nil {
		return object.NewError(0, "websocket connection is closed")
	}
	if err := c.conn.WriteMessage(messageType, payload); err != nil {
		c.closed = true
		return object.NewError(0, "could not write websocket message: %s", err.Error())
	}
	return &object.Null{}
}

func (c *nativeWebSocketConn) readMessage() (int, []byte, *object.Error) {
	c.mu.Lock()
	if c.closed || c.conn == nil {
		c.mu.Unlock()
		return 0, nil, nil
	}
	conn := c.conn
	c.mu.Unlock()

	messageType, data, err := conn.ReadMessage()
	if err != nil {
		if gws.IsCloseError(err, gws.CloseNormalClosure, gws.CloseGoingAway, gws.CloseNoStatusReceived) || strings.Contains(err.Error(), "close") {
			c.close()
			return 0, nil, nil
		}
		c.close()
		return 0, nil, object.NewError(0, "could not read websocket message: %s", err.Error())
	}
	return messageType, data, nil
}

func (c *nativeWebSocketConn) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.conn == nil {
		c.closed = true
		return
	}
	_ = c.conn.WriteControl(gws.CloseMessage, gws.FormatCloseMessage(gws.CloseNormalClosure, ""), time.Now().Add(100*time.Millisecond))
	_ = c.conn.Close()
	c.closed = true
	c.conn = nil
}

func (c *nativeWebSocketConn) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed || c.conn == nil
}

func writeWebSocketHandlerResult(conn *nativeWebSocketConn, result object.Object) *object.Error {
	switch value := result.(type) {
	case nil, *object.Null:
		return nil
	case *object.String:
		if errObj, ok := conn.writeMessage(gws.TextMessage, []byte(value.Value)).(*object.Error); ok {
			return errObj
		}
		return nil
	default:
		payload, err := json.Marshal(nativeValue(result))
		if err != nil {
			return object.NewError(0, "could not encode websocket handler result: %s", err.Error())
		}
		if errObj, ok := conn.writeMessage(gws.TextMessage, payload).(*object.Error); ok {
			return errObj
		}
		return nil
	}
}

func websocketMessageTypeName(messageType int) string {
	switch messageType {
	case gws.TextMessage:
		return "text"
	case gws.BinaryMessage:
		return "binary"
	case gws.CloseMessage:
		return "close"
	case gws.PingMessage:
		return "ping"
	case gws.PongMessage:
		return "pong"
	default:
		return "unknown"
	}
}
