package robo

import (
	"net/http"
	"net/url"
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

	// parsed querystring values
	query url.Values

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

// Query returns the value of a particular querystring parameter, after
// lazily parsing the raw querystring.
func (r *Request) Query(name string) string {
	if r.query == nil {
		r.query = r.URL.Query()
	}
	return r.query.Get(name)
}

// Param returns the value of a named URL parameter.
func (r *Request) Param(name string) string {
	return r.params[name]
}

// Mux is a HTTP router. It multiplexes incoming requests to different
// handlers based on user-provided rules on methods and paths.
//
// The zero value for a Mux is a Mux without any registered handlers,
// ready to use.
type Mux struct {
	routes []route
}

// NewMux returns a new Mux instance.
func NewMux() *Mux {
	return new(Mux)
}

// Any registers a new set of handlers listening to all requests for
// the specified URL pattern.
func (m *Mux) Any(pattern string, handlers ...interface{}) {
	m.add("", pattern, handlers...)
}

// Get registers a new set of handlers listening to GET requests for
// the specified URL pattern.
func (m *Mux) Get(pattern string, handlers ...interface{}) {
	m.add("GET", pattern, handlers...)
}

// Head registers a new set of handlers listening to HEAD requests for
// the specified URL pattern.
func (m *Mux) Head(pattern string, handlers ...interface{}) {
	m.add("HEAD", pattern, handlers...)
}

// Post registers a new set of handlers listening to POST requests for
// the specified URL pattern.
func (m *Mux) Post(pattern string, handlers ...interface{}) {
	m.add("POST", pattern, handlers...)
}

// Put registers a new set of handlers listening to PUT requests for
// the specified URL pattern.
func (m *Mux) Put(pattern string, handlers ...interface{}) {
	m.add("PUT", pattern, handlers...)
}

// Patch registers a new set of handlers listening to PATCH requests for
// the specified URL pattern.
func (m *Mux) Patch(pattern string, handlers ...interface{}) {
	m.add("PATCH", pattern, handlers...)
}

// Delete registers a new set of handlers listening to DELETE requests for
// the specified URL pattern.
func (m *Mux) Delete(pattern string, handlers ...interface{}) {
	m.add("DELETE", pattern, handlers...)
}

// Options registers a new set of handlers listening to OPTIONS requests for
// the specified URL pattern.
func (m *Mux) Options(pattern string, handlers ...interface{}) {
	m.add("OPTIONS", pattern, handlers...)
}

// add registers a set of handlers for the given HTTP method ("" matching
// any method) and URL pattern.
func (m *Mux) add(method, pattern string, handlers ...interface{}) {
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

// newRoute initializes a new route.
func newRoute(method, pattern string, handlers []Handler) route {
	matcher, err := compileMatcher(pattern)
	if err != nil {
		panic(err)
	}

	return route{method, matcher, handlers}
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
	matcher  pathMatcher
	handlers []Handler
}

var emptyParams = make(map[string]string)

// check tests whether the route matches a provided method and path. The
// parameter map will always be non-nil when the first is true.
func (r *route) check(method, path string) (bool, map[string]string) {
	if method != r.method && r.method != "" {
		return false, nil
	}

	ok, list := r.matcher.match(path, nil)
	if !ok {
		return false, nil
	}

	// don't build the actual parameter map unless we have to
	if len(list) == 0 {
		return true, emptyParams
	}

	params := make(map[string]string)
	for i := 0; i < len(list); i += 2 {
		params[list[i]] = list[i+1]
	}

	return true, params
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

		h.ServeRoboHTTP(w, &Request{hr, nil, q.params, q})
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
		r.handlers[0].ServeRoboHTTP(w, &Request{hr, nil, q.params, q})
		return
	}

	// when we run out of routes, send a 404 message
	http.Error(w, "Not found.\n", 404)
}
