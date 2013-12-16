package robo

import (
	"errors"
)

// Definitions of all possible pattern compilation errors.
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

// A pattern is a compiled URL pattern matcher.
type pattern []fragment

// match returns true as well as a slice of captured parameters in the form
// of [key, value] pairs (appended to buf), if the input string matches the
// pattern.
func (p pattern) match(in string, buf []string) (bool, []string) {
	var n int

	// check each fragment in order
	for _, f := range p {
		if in == "" {
			return false, nil
		}

		n, buf = f.match(in, buf)
		if n == 0 {
			return false, nil
		}

		in = in[n:]
	}

	// make sure the whole string has been consumed
	if in != "" {
		return false, nil
	}

	return true, buf
}

// The segment type describes a rule to be used when matching a
// subset of a pattern.
type fragment struct {
	t int
	s string
	r []rune
}

// match is the fragment-level equivalent of pattern.match.
func (f *fragment) match(in string, buf []string) (int, []string) {
	switch f.t {
	// literal fragment
	case 0:
		if len(in) < len(f.s) {
			return 0, nil
		}
		for i := 0; i < len(f.s); i++ {
			if f.s[i] != in[i] {
				return 0, nil
			}
		}
		return len(f.s), buf

	// exclusive parameter fragment
	case 1:
		for i, r := range in {
			for _, e := range f.r {
				if r == e {
					return i, append(buf, f.s, in[:i])
				}
			}
		}
		return len(in), append(buf, f.s, in)

	// include parameter fragment
	case 2:
		for i, r := range in {
			for j := 0; j < len(f.r); j += 2 {
				if f.r[j] <= r && r <= f.r[j+1] {
					goto ok
				}
			}
			return i, append(buf, f.s, in[:i])
		ok:
		}
		return len(in), append(buf, f.s, in)

	// wildcard fragment
	case 3:
		return len(in), append(buf, "*", in)
	}

	panic("unreachable")
}

// compilePattern compiles a pattern (as a sequence of fragments) according
// to the inputted format string.
func compilePattern(in string) (pattern, error) {
	var i int
	var o pattern

	if in == "" {
		return nil, errEmptyPattern
	}

	for i < len(in) {
		f, n, err := readFragment(in[i:])
		if err != nil {
			return nil, err
		}

		o = append(o, f)
		i += n
	}

	return o, nil
}

func readFragment(in string) (fragment, int, error) {
	switch in[0] {
	default:
		return readLiteral(in)
	case '*':
		return readWildcard(in)
	case '{':
		return readParameter(in)
	}
}

func readLiteral(in string) (fragment, int, error) {
	var i int
	var e bool

	for i = 0; i < len(in); i++ {
		switch c := in[i]; {
		case e:
			e = false
		case c == '\\':
			e = true
		case c == '*', c == '{':
			return fragment{t: 0, s: in[:i]}, i, nil
		}
	}

	return fragment{t: 0, s: in[:i]}, i, nil
}

func readWildcard(in string) (fragment, int, error) {
	if in != "*" {
		return fragment{}, 0, errIllegalWildcard
	}
	return fragment{t: 3}, 1, nil
}

func readParameter(in string) (fragment, int, error) {
	var i int
	var e bool

	for i = 1; i < len(in); i++ {
		switch c := in[i]; {
		case e:
			e = false

		case c == '\\':
			e = true

		case c == '[':
			chars, n, err := readCharset(in[i:])
			if err != nil {
				return fragment{}, 0, err
			}

			// before returning, make sure the charset block is
			// followed by a '}'
			if i := i + n; i == len(in) || in[i] != '}' {
				return fragment{}, 0, errMissingRBrace
			}

			return fragment{t: 2, s: in[1:i], r: chars}, i + n + 1, nil

		case c == '}':
			if i == 1 {
				return fragment{}, 0, errEmptyParameter
			}

			// if available, add the next rune to exclusive parameters (for
			// example: '-' for {foo} in "/{foo}-bar")
			for _, r := range in[i+1:] {
				if r == '/' {
					break
				}
				return fragment{t: 1, s: in[1:i], r: []rune{r, '/'}}, i + 1, nil
			}

			return fragment{t: 1, s: in[1:i], r: []rune{'/'}}, i + 1, nil
		}
	}

	return fragment{}, 0, errMissingRBrace
}

func readCharset(in string) ([]rune, int, error) {
	var o []rune
	var e bool

	// begin with some rudimentary error checking
	if len(in) < 2 {
		return nil, 0, errMissingRBracket
	} else if in[1] == ']' {
		return nil, 0, errEmptyCharset
	}

loop:
	for i, r := range in[1:] {
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

func simplifyCharset(a []rune) []rune {
	if len(a) == 0 {
		return a
	}

	// sort the pairs in the charset
	for i := 2; i < len(a); i += 2 {
		for j := i; j > 0; j -= 2 {
			if a[j] > a[j-2] || (a[j] == a[j-2] && a[j] > a[j-2]) {
				break
			}
			a[j-2], a[j+0] = a[j+0], a[j-2]
			a[j-1], a[j+1] = a[j+1], a[j-1]
		}
	}

	// merge overlapping pairs
	r := 2
	w := 2

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
