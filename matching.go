package robo

import (
	"errors"
)

var (
	errEmptyPattern   = errors.New("robo: empty pattern")
	errEmptyParameter = errors.New("robo: empty parameter name")
	errEmptyCharset   = errors.New("robo: empty charset")

	errCharsetHasSlash = errors.New("robo: parameter charset includes '/'")
	errIllegalWildcard = errors.New("robo: illegal '*' position")
	errImpossibleRange = errors.New("robo: impossible charset range")

	errUnexpectedHyphen   = errors.New("robo: unexpected '-'")
	errUnexpectedLBracket = errors.New("robo: unexpected '['")
	errUnexpectedRBracket = errors.New("robo: unexpected ']'")
	errMissingRBrace      = errors.New("robo: missing closing '}'")
	errMissingRBracket    = errors.New("robo: missing closing ']'")
)

// The pathMatcher interface is used to match the paths of incoming requests.
// Any captured parameters will be appended to buf and returned as the second
// return value.
type pathMatcher interface {
	match(path string, buf []string) (bool, []string)
}

// compileMatcher compiles a pathMatcher from a pattern string.
func compileMatcher(pattern string) (pathMatcher, error) {
	var fs []fragment

	if pattern == "" {
		return nil, errEmptyPattern
	}

	for pattern != "" {
		f, n, err := compileFragment(pattern)
		if err != nil {
			return nil, err
		}

		fs = append(fs, f)
		pattern = pattern[n:]
	}

	return &fragmentMatcher{fs}, nil
}

// compileFragment compiles a fragment matcher from a prefix of a pattern
// string. It returns the compiled fragment, and how many bytes of the input
// string were consumed.
func compileFragment(pattern string) (fragment, int, error) {
	switch pattern[0] {
	default:
		return compileLiteralFragment(pattern)
	case '*':
		return compileWildcardFragment(pattern)
	case '{':
		return compileParameterFragment(pattern)
	}
}

func compileLiteralFragment(pattern string) (fragment, int, error) {
	var i int
	var e bool

	for i = 0; i < len(pattern); i++ {
		switch c := pattern[i]; {
		case e:
			e = false
		case c == '\\':
			e = true
		case c == '*', c == '{':
			return fragment{t: literalFragment, s: pattern[:i], n: i}, i, nil
		}
	}

	return fragment{t: literalFragment, s: pattern[:i], n: i}, i, nil
}

func compileWildcardFragment(pattern string) (fragment, int, error) {
	if pattern != "*" {
		return fragment{}, 0, errIllegalWildcard
	}
	return fragment{t: wildcardFragment}, 1, nil
}

func compileParameterFragment(pattern string) (fragment, int, error) {
	var i int
	var e bool
	var f fragment

	for i = 1; i < len(pattern); i++ {
		switch c := pattern[i]; {
		case e:
			e = false

		case c == '\\':
			e = true

		case c == '[':
			chars, n, err := compileCharsetFragment(pattern[i:])
			if err != nil {
				return f, 0, err
			}

			// before returning, make sure the charset block is
			// followed by a '}'
			if i := i + n; i == len(pattern) || pattern[i] != '}' {
				return fragment{}, 0, errMissingRBrace
			}

			f = fragment{t: inclusiveFragment, s: pattern[1:i], r: chars}
			return f, i + n + 1, nil

		case c == '}':
			if i == 1 {
				return fragment{}, 0, errEmptyParameter
			}

			f = fragment{t: exclusiveFragment, s: pattern[1:i], r: []rune{'/'}}

			// if available, add the next rune to exclusive parameters (for
			// example: '-' for {foo} in "/{foo}-bar")
			for _, r := range pattern[i+1:] {
				if r != '/' {
					f.r = append(f.r, r)
				}
				break
			}

			return f, i + 1, nil
		}
	}

	return f, 0, errMissingRBrace
}

func compileCharsetFragment(pattern string) ([]rune, int, error) {
	var o []rune
	var e bool

	// begin with some rudimentary error checking
	if len(pattern) < 2 {
		return nil, 0, errMissingRBracket
	} else if pattern[1] == ']' {
		return nil, 0, errEmptyCharset
	}

loop:
	for i, r := range pattern[1:] {
		if !e {
			switch r {
			case '\\':
				e = true
				continue loop

			case '[':
				return nil, 0, errUnexpectedLBracket

			case ']':
				if len(o)&1 != 0 {
					return nil, 0, errUnexpectedRBracket
				}

				// make sure the charset doesn't include forward slash
				for i := 0; i < len(o); i += 2 {
					if o[i] <= '/' && '/' <= o[i+1] {
						return nil, 0, errCharsetHasSlash
					}
				}

				return simplifyCharset(o), i + 2, nil

			case '-':
				// catch ambiguous range statements like "a-c-e"
				if len(o) == 0 || o[len(o)-1] != o[len(o)-2] {
					return nil, 0, errUnexpectedHyphen
				}

				o = o[:len(o)-1]
				continue loop
			}
		}

		// determine whether or not this character is the upper bound
		// of a charset range
		if len(o)&1 != 0 {
			if r < o[len(o)-1] {
				return nil, 0, errImpossibleRange
			}
			o = append(o, r)
		} else {
			o = append(o, r, r)
		}

		e = false
	}

	return nil, 0, errMissingRBracket
}

// simplifyCharset merges overlapping rune ranges in the input charset.
func simplifyCharset(a []rune) []rune {
	if len(a) == 0 {
		return a
	}

	// first, sort the pairs in the charset
	for i := 2; i < len(a); i += 2 {
		for j := i; j > 0; j -= 2 {
			if a[j] > a[j-2] || (a[j] == a[j-2] && a[j] > a[j-2]) {
				break
			}
			a[j-2], a[j+0] = a[j+0], a[j-2]
			a[j-1], a[j+1] = a[j+1], a[j-1]
		}
	}

	// then merge overlapping pairs
	var r, w = 2, 2

	for ; r < len(a); r += 2 {
		if a[r] <= a[w-2] {
			if a[w-2] <= a[r+1]+1 {
				goto merge
			}
		} else if a[r]-1 <= a[w-1] {
			goto merge
		}

		a[w] = a[r]
		a[w+1] = a[r+1]
		w += 2
		continue

	merge:
		if a[r] < a[w-2] {
			a[w-2] = a[r]
		}
		if a[r+1] > a[w-1] {
			a[w-1] = a[r+1]
		}
	}

	return a[:w]
}

// fragmentMatcher is an implementation of the pathMatcher interface,
// which matches input strings using precompiled fragments.
type fragmentMatcher struct {
	fs []fragment
}

func (f *fragmentMatcher) match(path string, buf []string) (bool, []string) {
	var n int

	for _, f := range f.fs {
		n, buf = f.matchPrefix(path, buf)
		if n < 0 {
			return false, nil
		}
		path = path[n:]
	}

	// make sure the whole string has been consumed
	if path != "" {
		return false, nil
	}

	return true, buf
}

type fragment struct {
	t int
	s string
	n int
	r []rune
}

const (
	literalFragment = iota
	exclusiveFragment
	inclusiveFragment
	wildcardFragment
)

func (f *fragment) matchPrefix(pattern string, buf []string) (int, []string) {
	switch f.t {
	case literalFragment:
		if len(pattern) < f.n || pattern[:f.n] != f.s {
			return -1, nil
		}
		return f.n, buf

	case exclusiveFragment:
		for i, r := range pattern {
			for _, excl := range f.r {
				if r == excl {
					return nonZero(i), append(buf, f.s, pattern[:i])
				}
			}
		}
		return nonZero(len(pattern)), append(buf, f.s, pattern)

	case inclusiveFragment:
		for i, r := range pattern {
			for j := 0; j < len(f.r); j += 2 {
				if f.r[j] <= r && r <= f.r[j+1] {
					goto ok
				}
			}
			return nonZero(i), append(buf, f.s, pattern[:i])
		ok:
		}
		return nonZero(len(pattern)), append(buf, f.s, pattern)

	case wildcardFragment:
		return len(pattern), append(buf, "*", pattern)
	}

	panic("unreachable")
}

// nonZero return -1 instead of n if n == 0.
func nonZero(n int) int {
	if n == 0 {
		return -1
	}
	return n
}
