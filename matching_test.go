package robo

import (
	"testing"
)

type matcherCheck struct {
	pattern string
	ok      bool
	params  []string
}

var matcherTests = []struct {
	format string
	err    error
	checks []matcherCheck
}{
	{"/", nil, []matcherCheck{
		{"/", true, nil},
		{"//", false, nil},
		{"/foo", false, nil},
	}},
	{"/foo", nil, []matcherCheck{
		{"/", false, nil},
		{"/foo", true, nil},
		{"/foo/", false, nil},
		{"/foo/bar", false, nil},
	}},
	{"*", nil, []matcherCheck{
		{"/", true, []string{"*", "/"}},
		{"/foo", true, []string{"*", "/foo"}},
		{"/foo/bar/", true, []string{"*", "/foo/bar/"}},
	}},
	{"/foo/*", nil, []matcherCheck{
		{"/", false, nil},
		{"/foo", false, nil},
		{"/foo/", true, []string{"*", ""}},
		{"/foo/bar", true, []string{"*", "bar"}},
		{"/foo/bar/qux", true, []string{"*", "bar/qux"}},
	}},
	{"/{foo}", nil, []matcherCheck{
		{"/", false, nil},
		{"/fo", true, []string{"foo", "fo"}},
		{"/foo", true, []string{"foo", "foo"}},
		{"/fooo", true, []string{"foo", "fooo"}},
		{"/foo-bar", true, []string{"foo", "foo-bar"}},
		{"/foo/bar", false, nil},
	}},
	{"/{foo[a-z]}", nil, []matcherCheck{
		{"/", false, nil},
		{"/foo", true, []string{"foo", "foo"}},
		{"/f00", false, nil},
		{"/foo/bar", false, nil},
	}},
	{"/{foo}-{bar}", nil, []matcherCheck{
		{"/", false, nil},
		{"/foo-bar", true, []string{"foo", "foo", "bar", "bar"}},
		{"/foo-", false, nil},
		{"/f00", false, nil},
		{"/foo/bar", false, nil},
	}},
	{"/{foo[a-z]}{bar[0-9]}", nil, []matcherCheck{
		{"/", false, nil},
		{"/foo123", true, []string{"foo", "foo", "bar", "123"}},
		{"/f1", true, []string{"foo", "f", "bar", "1"}},
		{"/foo", false, nil},
		{"/123", false, nil},
	}},

	{"", errEmptyPattern, nil},
	{"/*/foo", errIllegalWildcard, nil},
	{"/{foo", errMissingRBrace, nil},
	{"/{foo[]}", errEmptyCharset, nil},
	{"/{foo[}", errMissingRBracket, nil},
	{"/{foo[\\]}", errMissingRBracket, nil},
	{"/{foo[a-]}", errUnexpectedRBracket, nil},
	{"/{foo[abc[]}", errUnexpectedLBracket, nil},
	{"/{foo[z-a]}", errImpossibleRange, nil},
	{"/{foo[a-b-c]}", errUnexpectedHyphen, nil},
}

func TestMatcher(t *testing.T) {
	for _, test := range matcherTests {
		matcher, err := compileMatcher(test.format)
		if err != test.err {
			t.Errorf("compileMatcher(%q):", test.format)
			t.Errorf("  got  %v", err)
			t.Errorf("  want %v", test.err)
			continue
		}

		for _, check := range test.checks {
			ok, params := matcher.match(check.pattern, nil)
			if ok != check.ok || len(params) != len(check.params) {
				goto fail
			}

			for i := range params {
				if params[i] != check.params[i] {
					goto fail
				}
			}

			continue

		fail:
			t.Errorf("%+v.match(%q):", matcher, check.pattern)
			t.Errorf("  got  %v, %+v", ok, params)
			t.Errorf("  want %v, %+v", check.ok, check.params)
		}
	}
}

var simplifyCharsetTests = []struct {
	input  []rune
	output []rune
}{
	{[]rune{'a', 'z'}, []rune{'a', 'z'}},
	{[]rune{'a', 'a', 'b', 'b', 'c', 'c', 'd', 'd'}, []rune{'a', 'd'}},
	{[]rune{'a', 'a', 'c', 'c', 'd', 'd'}, []rune{'a', 'a', 'c', 'd'}},
	{[]rune{'a', 'b', 'c', 'd', 'e', 'f'}, []rune{'a', 'f'}},
	{[]rune{'a', 'f', 'b', 'c', 'd', 'e'}, []rune{'a', 'f'}},
}

func TestSimplifyCharset(t *testing.T) {
	for _, test := range simplifyCharsetTests {
		dup := make([]rune, len(test.input))
		copy(dup, test.input)

		output := simplifyCharset(dup)
		if len(output) != len(test.output) {
			goto fail
		}

		for i := range output {
			if output[i] != test.output[i] {
				goto fail
			}
		}

		continue

	fail:
		t.Errorf("simplifyCharset(%q):", test.input)
		t.Errorf("   got  %q", output)
		t.Errorf("   want %q", test.output)
	}
}
