package robo

import (
	"net/http"
	"net/url"
)

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
