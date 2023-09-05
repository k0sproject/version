package version

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var constraintRegex = regexp.MustCompile(`^(?:(>=|>|<=|<|!=|==?)\s*)?(.+)$`)

type constraintFunc func(a, b *Version) bool
type constraint struct {
	f        constraintFunc
	b        *Version
	original string
}

// Constraints is a collection of version constraint rules that can be checked against a version.
type Constraints []constraint

// NewConstraint parses a string into a Constraints object that can be used to check
// if a given version satisfies the constraint.
func NewConstraint(cs string) (Constraints, error) {
	parts := strings.Split(cs, ",")
	newC := make(Constraints, len(parts))
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	for i, p := range parts {
		c, err := newConstraint(p)
		if err != nil {
			return Constraints{}, err
		}
		newC[i] = c
	}

	return newC, nil
}

// MustConstraint is like NewConstraint but panics if the constraint is invalid.
func MustConstraint(cs string) Constraints {
	c, err := NewConstraint(cs)
	if err != nil {
		panic("github.com/k0sproject/version: NewConstraint: " + err.Error())
	}
	return c
}

// String returns the constraint as a string.
func (cs Constraints) String() string {
	s := make([]string, len(cs))
	for i, c := range cs {
		s[i] = c.String()
	}
	return strings.Join(s, ", ")
}

// Check returns true if the given version satisfies all of the constraints.
func (cs Constraints) Check(v *Version) bool {
	for _, c := range cs {
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
func (cs Constraints) CheckString(v string) bool {
	vv, err := NewVersion(v)
	if err != nil {
		return false
	}
	return cs.Check(vv)
}

// String returns the original constraint string.
func (c *constraint) String() string {
	return c.original
}

func newConstraint(s string) (constraint, error) {
	match := constraintRegex.FindStringSubmatch(s)
	if len(match) != 3 {
		return constraint{}, errors.New("invalid constraint: " + s)
	}

	op := match[1]
	f, err := opfunc(op)
	if err != nil {
		return constraint{}, err
	}

	// convert one or two digit constraints to threes digit unless it's an equality operation
	if op != "" && op != "=" && op != "==" {
		segments := strings.Split(match[2], ".")
		if len(segments) < 3 {
			lastSegment := segments[len(segments)-1]
			var pre string
			if strings.Contains(lastSegment, "-") {
				parts := strings.Split(lastSegment, "-")
				segments[len(segments)-1] = parts[0]
				pre = "-" + parts[1]
			}
			switch len(segments) {
			case 1:
				// >= 1 becomes >= 1.0.0
				// >= 1-rc.1 becomes >= 1.0.0-rc.1
				return newConstraint(fmt.Sprintf("%s %s.0.0%s", op, segments[0], pre))
			case 2:
				// >= 1.1 becomes >= 1.1.0
				// >= 1.1-rc.1 becomes >= 1.1.0-rc.1
				return newConstraint(fmt.Sprintf("%s %s.%s.0%s", op, segments[0], segments[1], pre))
			}
		}
	}

	target, err := NewVersion(match[2])
	if err != nil {
		return constraint{}, err
	}

	return constraint{f: f, b: target, original: s}, nil
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

