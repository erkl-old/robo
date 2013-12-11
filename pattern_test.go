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
