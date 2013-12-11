package robo

// The segment type performs string matching on a part of a pattern.
type segment struct {
	t int
	s string
	r []rune
}

// Match looks for a match at the beginning of the input string, returning the
// size of the match in bytes.
func (s *segment) Match(in string) int {
	switch s.t {
	// literal prefix
	case 0:
		if len(in) < len(s.s) {
			return 0
		}
		for i := 0; i < len(s.s); i++ {
			if s.s[i] != in[i] {
				return 0
			}
		}
		return len(s.s)

	// wildcard parameter
	case 1:
		for i, r := range in {
			if r == '/' {
				return i
			}
		}
		return len(in)

	// charset parameter
	case 2:
		for i, r := range in {
			if r == '/' {
				return i
			}
			for j := 0; j < len(s.r); j += 2 {
				if s.r[j] <= r && r <= s.r[j+1] {
					goto ok
				}
			}
			return 0
		ok:
		}
		return len(in)

	// wildcard affix
	case 3:
		return len(in)
	}

	panic("unreachable")
}

// Name returns the parameter name of the segment.
func (s *segment) Name() string {
	if s.t == 1 || s.t == 2 {
		return s.s
	}
	return ""
}
