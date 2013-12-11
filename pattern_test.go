package robo

import (
	"testing"
)

var segmentTests = []struct {
	s  segment
	in string
	n  int
}{
	{segment{0, "/foo", nil}, "/foo", 4},
	{segment{0, "/foo", nil}, "/bar", 0},
	{segment{0, "/foo", nil}, "/foo/bar", 4},
	{segment{0, "/foo/", nil}, "/foo/bar", 5},

	{segment{1, "", nil}, "foo", 3},
	{segment{1, "", nil}, "foo/bar", 3},
	{segment{1, "", nil}, "/bar", 0},

	{segment{2, "", []rune{'a', 'z'}}, "foo", 3},
	{segment{2, "", []rune{'a', 'z'}}, "foo/", 3},
	{segment{2, "", []rune{'a', 'z'}}, "/foo", 0},
	{segment{2, "", []rune{'a', 'z', '0', '9'}}, "1a23foo", 7},

	{segment{3, "", nil}, "foo", 3},
	{segment{3, "", nil}, "foo/bar", 7},
	{segment{3, "", nil}, "/a/b/c//", 8},
}

func TestSegmentMatch(t *testing.T) {
	for _, test := range segmentTests {
		if n := test.s.Match(test.in); n != test.n {
			t.Errorf("%+v.Match(%q)\n", test.s, test.in)
			t.Errorf("  got  %d\n", n)
			t.Errorf("  want %d\n", test.n)
		}
	}
}

var patternTests = []struct {
	p  pattern
	in string
	ok bool
	m  map[string]string
}{
	{[]segment{segment{0, "/", nil}}, "/", true, nil},
	{[]segment{segment{0, "/foo/", nil}, segment{3, "", nil}}, "/foo/bar", true, nil},
	{[]segment{segment{0, "/foo/", nil}, segment{3, "", nil}}, "/foo-bar", false, nil},
	{[]segment{segment{0, "/", nil}, segment{1, "a", nil}}, "/foo", true, map[string]string{"a": "foo"}},
	{[]segment{segment{0, "/", nil}, segment{1, "b", nil}}, "/foo-bar", true, map[string]string{"b": "foo-bar"}},
	{[]segment{segment{0, "/", nil}, segment{1, "b", nil}}, "/foo/bar", false, nil},
	{[]segment{segment{0, "/", nil}, segment{1, "a", nil}, segment{0, "/", nil}, segment{2, "b", []rune{'a', 'z'}}}, "/foo/bar", true, map[string]string{"a": "foo", "b": "bar"}},
	{[]segment{segment{0, "/", nil}, segment{1, "a", nil}, segment{0, "/", nil}, segment{2, "b", []rune{'a', 'z'}}}, "/foo/123", false, nil},
}

func TestPatternMatch(t *testing.T) {
	for _, test := range patternTests {
		ok, m := test.p.Match(test.in)
		if ok != test.ok || len(m) != len(test.m) {
			goto fail
		}

		for key := range m {
			if m[key] != test.m[key] {
				goto fail
			}
		}

		continue

	fail:
		t.Errorf("%+v.Match(%q):", test.p, test.in)
		t.Errorf("  want %v, %+v", test.ok, test.m)
		t.Errorf("  want %v, %+v", ok, m)
	}
}
