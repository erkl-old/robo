package robo

import (
	"net/http"
	"net/url"
)

// The Request type extends an http.Request instance with additional
// functionality.
type Request struct {
	*http.Request

	// parsed querystring values (lazily generated)
	query url.Values

	// named URL parameters, specific to the route
	params map[string]string

	// pointer to the request-local data map, which is stored in the
	// queue and shared between all routes
	store **map[string]interface{}

	// reference to the request's queue, used by the Next method
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

// Get returns a value stored in the request's data store (or nil if
// it hasn't been defined yet).
func (r *Request) Get(key string) interface{} {
	if *r.store == nil {
		return nil
	}
	return (**r.store)[key]
}

// Set stores a value in the request's data store.
func (r *Request) Set(key string, value interface{}) {
	if *r.store == nil {
		m := make(map[string]interface{})
		*r.store = &m
	}
	(**r.store)[key] = value
}
