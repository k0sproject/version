package version

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var constraintRegex = regexp.MustCompile(`^\s*(?:(>=|>|<=|<|!=|==?)\s*)?(.+)\s*$`)

type (
	constraintFunc    func(a, b *Version) bool
	constraintSegment struct {
		f constraintFunc
		b *Version
	}
	Constraint struct {
		segments []constraintSegment
		original string
	}
)

// NewConstraint parses a string and returns a Contraint and an error if the parsing fails.
func NewConstraint(cs string) (Constraint, error) {
	c := Constraint{original: cs}
	parts := strings.Split(cs, ",")
	for _, p := range parts {
		segments, err := parseSegment(p)
		if err != nil {
			return c, err
		}
		c.segments = append(c.segments, segments...)
	}

	return c, nil
}

// MustConstraint is like NewConstraint but panics if the constraint is invalid.
func MustConstraint(cs string) Constraint {
	c, err := NewConstraint(cs)
	if err != nil {
		panic("github.com/k0sproject/version: NewConstraint: " + err.Error())
	}
	return c
}

// String returns the constraint as a string.
func (cs Constraint) String() string {
	return cs.original
}

// Check returns true if the given version satisfies all of the constraints.
func (cs Constraint) Check(v *Version) bool {
	for _, c := range cs.segments {
		if c.b.Prerelease() == "" && v.Prerelease() != "" {
			return false
		}
		if !c.f(c.b, v) {
			return false
		}
	}

	return true
}

// CheckString is like Check but takes a string version. If the version is invalid,
// it returns false.
func (cs Constraint) CheckString(v string) bool {
	vv, err := NewVersion(v)
	if err != nil {
		return false
	}
	return cs.Check(vv)
}

func parseSegment(s string) ([]constraintSegment, error) {
	match := constraintRegex.FindStringSubmatch(s)
	if len(match) != 3 {
		return nil, errors.New("invalid constraint: " + s)
	}

	op := match[1]
	f, err := opfunc(op)
	if err != nil {
		return nil, err
	}

	// convert one or two digit constraints to threes digit unless it's an equality operation
	if op != "" && op != "=" && op != "==" {
		vSegments := strings.Split(match[2], ".")
		if len(vSegments) < 3 {
			lastSegment := vSegments[len(vSegments)-1]
			var pre string
			if strings.Contains(lastSegment, "-") {
				parts := strings.Split(lastSegment, "-")
				vSegments[len(vSegments)-1] = parts[0]
				pre = "-" + parts[1]
			}
			switch len(vSegments) {
			case 1:
				// >= 1 becomes >= 1.0.0
				// >= 1-rc.1 becomes >= 1.0.0-rc.1
				return parseSegment(fmt.Sprintf("%s %s.0.0%s", op, vSegments[0], pre))
			case 2:
				// >= 1.1 becomes >= 1.1.0
				// >= 1.1-rc.1 becomes >= 1.1.0-rc.1
				return parseSegment(fmt.Sprintf("%s %s.%s.0%s", op, vSegments[0], vSegments[1], pre))
			}
		}
	}

	target, err := NewVersion(match[2])
	if err != nil {
		return nil, err
	}

	return []constraintSegment{{f: f, b: target}}, nil
}

func opfunc(s string) (constraintFunc, error) {
	switch s {
	case "", "=", "==":
		return eq, nil
	case ">":
		return gt, nil
	case ">=":
		return gte, nil
	case "<":
		return lt, nil
	case "<=":
		return lte, nil
	case "!=":
		return neq, nil
	default:
		return nil, errors.New("invalid operator: " + s)
	}
}

func gt(a, b *Version) bool  { return b.GreaterThan(a) }
func lt(a, b *Version) bool  { return b.LessThan(a) }
func gte(a, b *Version) bool { return b.GreaterThanOrEqual(a) }
func lte(a, b *Version) bool { return b.LessThanOrEqual(a) }
func eq(a, b *Version) bool  { return b.Equal(a) }
func neq(a, b *Version) bool { return !b.Equal(a) }
