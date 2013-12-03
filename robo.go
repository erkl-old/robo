// Package robo provides a tiny HTTP server framework.
package robo

import (
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// Mux is a HTTP router implementing the http.Handler interface.
type Mux struct {
	node
}

// NewMux initializes a new Mux instance.
func NewMux() *Mux {
	return &Mux{*newNode()}
}

// Add registers a handler for a particular request method and path pattern.
// The table below attempts to illustrate how patterns match URL paths of
// incoming requests:
//
//     patterns
//        ↓  paths → |  /foo/qux  |  /foo//qux  |  /foo/  |  /foo
//   ----------------+------------+-------------+---------+--------
//       /foo/:bar   |      ✔     |      ✔      |         |
//       /:x/qux     |      ✔     |      ✔      |         |
//       /foo/       |            |             |    ✔    |
//       /foo        |            |             |         |    ✔
//       /           |            |             |         |
//
// All matched portions of the incoming URL path are made available as synthetic
// querystring parameters. For example, when the pattern "/users/:id" matches
// the URL path "/users/123", r.URL.Query().Get(":id") will return "123".
func (m *Mux) Add(method, pattern string, handler http.Handler) {
	m.add(strings.ToUpper(method), split(pattern), &entry{h: handler})
}

// ServeHTTP routes an incoming request to a matching handler.
func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pattern := split(r.URL.Path)
	methods := make(map[string]*entry)

	m.search(pattern, methods)

	// not found
	if len(methods) == 0 {
		http.Error(w, "404", 404)
		return
	}

	ent := methods[r.Method]

	// method not allowed
	if ent == nil {
		allow := make([]string, 0, len(methods))
		for method := range methods {
			allow = append(allow, method)
		}

		sort.Strings(allow)

		w.Header().Set("Allow", strings.Join(allow, ", "))
		http.Error(w, "405", 405)
		return
	}

	// map captured URL parameters to synthetic querystring parameters
	params := make(url.Values)

	for i, val := range pattern {
		if frag := ent.p[i]; frag != "" {
			params.Set(frag, val)
		}
	}

	if r.URL.RawQuery != "" {
		r.URL.RawQuery = params.Encode() + "&" + r.URL.RawQuery
	} else {
		r.URL.RawQuery = params.Encode()
	}

	// finally, hand things over to the actual handler
	ent.h.ServeHTTP(w, r)
}

// nodes store request handlers in a trie structure.
type node struct {
	// map of HTTP methods to handler entries
	m map[string]*entry

	// child nodes, keyed by the next URL fragment to match
	c map[string]*node

	// URL parameter/wildcard child node
	w *node
}

type entry struct {
	// URL parameter names ("" if n/a)
	p []string

	// the handler in question
	h http.Handler
}

// newNode initializes a new node.
func newNode() *node {
	return &node{
		m: make(map[string]*entry),
		c: make(map[string]*node),
	}
}

// add registers a new handler.
func (n *node) add(method string, path []string, ent *entry) {
	if len(path) == 0 {
		n.m[method] = ent
		return
	}

	key := path[0]

	// URL parameters have to be treated separately
	if len(key) > 0 && key[0] == ':' {
		if len(key) == 1 {
			panic("invalid w")
		}

		if n.w == nil {
			n.w = newNode()
		}

		ent.p = append(ent.p, key)
		n.w.add(method, path[1:], ent)
	} else {
		child := n.c[key]

		if child == nil {
			child = newNode()
			n.c[key] = child
		}

		ent.p = append(ent.p, "")
		child.add(method, path[1:], ent)
	}
}

// search recursively searches the node and its children for handlers matching
// a sequence of path fragments. It returns a map of method names to handler
// entries.
func (n *node) search(path []string, out map[string]*entry) {
	if len(path) == 0 {
		for m, e := range n.m {
			if out[m] == nil {
				out[m] = e
			}
		}
		return
	}

	frag := path[0]

	// do we have a child node matching this path fragment?
	if child := n.c[frag]; child != nil {
		child.search(path[1:], out)
	}

	// also try the wildcard
	if frag != "" && n.w != nil {
		n.w.search(path[1:], out)
	}
}

// split separates an URL path to a sequence of fragments.
//
//      "/"          ->  {""}
//      "/foo"       ->  {"foo"}
//      "/foo/"      ->  {"foo", ""}
//      "/foo/bar"   ->  {"foo", "bar"}
//      "/foo/:bar"  ->  {"foo", ":bar"}
func split(s string) []string {
	path := strings.Split(s, "/")

	var i, j int
	for ; i < len(path); i++ {
		if path[i] != "" || i == len(path)-1 {
			path[j] = path[i]
			j++
		}
	}

	return path[:j]
}
