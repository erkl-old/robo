package robo

import (
	"net/http"
)

// Objects implementing the Handler interface are capable of serving
// HTTP requests. It is expected to follow the same core conventions as
// the "net/http" equivalent.
type Handler interface {
	ServeRoboHTTP(w ResponseWriter, r *Request)
}

// The HandlerFunc type serves as an adaptor to turn plain functions into
// an implementation of the Handler interface.
type HandlerFunc func(w ResponseWriter, r *Request)

func (h HandlerFunc) ServeRoboHTTP(w ResponseWriter, r *Request) {
	h(w, r)
}

// The httpHandler type adds a ServeRoboHTTP method to implementations of
// the http.Handler interface.
type httpHandler struct {
	h http.Handler
}

func (h httpHandler) ServeRoboHTTP(w ResponseWriter, r *Request) {
	h.h.ServeHTTP(w, r.Request)
}

// The ResponseWriter type mirrors http.ResponseWriter.
type ResponseWriter interface {
	http.ResponseWriter
}

// The Request type extends an http.Request instance with additional
// functionality.
type Request struct {
	*http.Request

	// named URL parameters for this request and route
	params map[string]string

	// reference to the queue
	queue *queue
}

// Next yields execution to the next matching handler, if there is one,
// blocking until said handler has returned.
func (r *Request) Next(w ResponseWriter) {
	r.queue.serveNext(w, r.Request)
}

// Param returns the value of a named URL parameter.
func (r *Request) Param(name string) string {
	if r.params != nil {
		return r.params[name]
	}
	return ""
}

// Mux is a HTTP router. It multiplexes incoming requests to different
// handlers based on user-provided rules on methods and paths.
//
// The zero value for a Mux is a Mux without any registered handlers,
// ready to use.
type Mux struct {
	routes []route
}

// Add registers a set of handlers for the given HTTP method and URL pattern.
//
// The following types are valid handler arguments:
//     robo.Handler
//     http.Handler
//     func(w robo.ResponseWriter, r *robo.Request)
//     func(w http.ResponseWriter, r *http.Request)
func (m *Mux) Add(method, pattern string, handlers ...interface{}) {
	if len(handlers) == 0 {
		panic("no handlers provided")
	}

	// validate the provided set of handlers
	clean := make([]Handler, 0, len(handlers))

	for _, h := range handlers {
		switch h := h.(type) {
		case Handler:
			clean = append(clean, h)
		case func(w ResponseWriter, r *Request):
			clean = append(clean, HandlerFunc(h))
		case http.Handler:
			clean = append(clean, httpHandler{h})
		case func(w http.ResponseWriter, r *http.Request):
			clean = append(clean, httpHandler{http.HandlerFunc(h)})
		default:
			panic("not a valid handler")
		}
	}

	m.routes = append(m.routes, newRoute(method, pattern, clean))
}

// ServeRoboHTTP dispatches the request to matching routes registered with
// the Mux instance.
func (m *Mux) ServeRoboHTTP(w ResponseWriter, r *Request) {
	q := queue{nil, nil, m.routes}
	q.serveNext(w, r.Request)
}

// ServeHTTP dispatches the request to matching routes registered with
// the Mux instance.
func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.ServeRoboHTTP(w, &Request{Request: r})
}

// The route type describes a registered route.
type route struct {
	method   string
	pattern  string
	handlers []Handler

	// @todo
}

// newRoute compiles a new route. Will panic() when the pattern contains
// a syntax error.
func newRoute(method, pattern string, handlers []Handler) route {
	r := route{
		method:   method,
		pattern:  pattern,
		handlers: handlers,
	}

	// @todo
	return r
}

// check tests whether the route matches a provided method and path. The
// parameter map will always be non-nil when the first is true.
func (r *route) check(method, path string) (bool, map[string]string) {
	// @todo
	return false, nil
}

// The queue type holds the routing state of an incoming request.
type queue struct {
	// remaining handlers, and parameter map, for the current route
	handlers []Handler
	params   map[string]string

	// remaining routes to be tested
	routes []route
}

// ServeNext attempts to serve an HTTP request using the next matching
// route/handler in the queue.
func (q *queue) serveNext(w ResponseWriter, hr *http.Request) {
	// does the current route still have handlers left?
	if len(q.handlers) > 0 {
		h := q.handlers[0]
		q.handlers = q.handlers[1:]

		h.ServeRoboHTTP(w, &Request{hr, q.params, q})
		return
	}

	// look for the next matching route
	for len(q.routes) > 0 {
		r := q.routes[0]
		q.routes = q.routes[1:]

		// does this route match the request at hand?
		ok, params := r.check(hr.Method, hr.URL.Path)
		if !ok {
			continue
		}

		q.handlers = r.handlers[1:]
		q.params = params

		// invoke the route's first handler
		r.handlers[0].ServeRoboHTTP(w, &Request{hr, q.params, q})
		return
	}
}
