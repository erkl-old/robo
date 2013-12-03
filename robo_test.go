package robo

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s - %s\n", r.URL.Path, r.URL.RawQuery)
}

var basicTests = []struct {
	method string
	url    string
	code   int
	out    string
}{
	{"GET", "http://example.com/foo", 200, "/foo - \n"},
	{"GET", "http://example.com/bar", 200, "/bar - %3Aany=bar\n"},
	{"GET", "http://example.com/users/123", 200, "/users/123 - %3Aid=123\n"},
	{"GET", "http://example.com/users/123?a=b", 200, "/users/123 - %3Aid=123&a=b\n"},
	{"GET", "http://example.com/users/", 404, "404\n"},
	{"GET", "http://example.com/users/?%3Aid=123", 404, "404\n"},
	{"GET", "http://example.com/users/foo/bar", 200, "/users/foo/bar - \n"},
	{"HEAD", "http://example.com/foo", 405, "405\n"},
}

func TestMux(t *testing.T) {
	mux := NewMux()

	mux.Add("GET", "/:any", http.HandlerFunc(handler))
	mux.Add("GET", "/foo", http.HandlerFunc(handler))
	mux.Add("GET", "/users/:id", http.HandlerFunc(handler))
	mux.Add("GET", "/users/foo/bar", http.HandlerFunc(handler))

	for _, test := range basicTests {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest(test.method, test.url, nil)

		mux.ServeHTTP(w, r)

		if out := w.Body.String(); out != test.out {
			t.Errorf("%q != %q", out, test.out)
		}
	}
}
