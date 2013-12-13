package robo

import (
	"testing"
)

type patternCheck struct {
	input  string
	ok     bool
	params []string
}

var patternTests = []struct {
	format string
	err    error
	checks []patternCheck
}{
	// legal
	{"/", nil, []patternCheck{
		{"/", true, nil},
		{"//", false, nil},
		{"/foo", false, nil},
	}},
	{"/foo", nil, []patternCheck{
		{"/", false, nil},
		{"/foo", true, nil},
		{"/foo/", false, nil},
		{"/foo/bar", false, nil},
	}},
	{"*", nil, []patternCheck{
		{"/", true, []string{"*", "/"}},
		{"/foo", true, []string{"*", "/foo"}},
		{"/foo/bar/", true, []string{"*", "/foo/bar/"}},
	}},
	{"/foo/*", nil, []patternCheck{
		{"/", false, nil},
		{"/foo", false, nil},
		{"/foo/bar", true, []string{"*", "bar"}},
		{"/foo/bar/qux", true, []string{"*", "bar/qux"}},
	}},
	{"/{foo}", nil, []patternCheck{
		{"/", false, nil},
		{"/fo", true, []string{"foo", "fo"}},
		{"/foo", true, []string{"foo", "foo"}},
		{"/fooo", true, []string{"foo", "fooo"}},
		{"/foo-bar", true, []string{"foo", "foo-bar"}},
		{"/foo/bar", false, nil},
	}},
	{"/{foo[a-z]}", nil, []patternCheck{
		{"/", false, nil},
		{"/foo", true, []string{"foo", "foo"}},
		{"/f00", false, nil},
		{"/foo/bar", false, nil},
	}},
	{"/{foo}-{bar}", nil, []patternCheck{
		{"/", false, nil},
		{"/foo-bar", true, []string{"foo", "foo", "bar", "bar"}},
		{"/foo-", false, nil},
		{"/f00", false, nil},
		{"/foo/bar", false, nil},
	}},
	{"/{foo[a-z]}{bar[0-9]}", nil, []patternCheck{
		{"/", false, nil},
		{"/foo123", true, []string{"foo", "foo", "bar", "123"}},
		{"/f1", true, []string{"foo", "f", "bar", "1"}},
		{"/foo", false, nil},
		{"/123", false, nil},
	}},

	// illegal
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

func TestPattern(t *testing.T) {
	for _, test := range patternTests {
		pattern, err := compilePattern(test.format)
		if err != test.err {
			t.Errorf("compilePattern(%q):", test.format)
			t.Errorf("  got  %v", err)
			t.Errorf("  want %v", test.err)
			continue
		}

		for _, check := range test.checks {
			ok, params := pattern.Match(check.input, nil)
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
			t.Errorf("%v.Match(%q):", pattern, check.input)
			t.Errorf("  got  %v, %+v", ok, params)
			t.Errorf("  want %v, %+v", check.ok, check.params)
		}
	}
}
