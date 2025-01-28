package version_test

import (
	"fmt"
	"testing"

	"github.com/k0sproject/version"
)

func TestCheck(t *testing.T) {
	type testCase struct {
		constraint string
		truthTable map[bool][]string
	}

	testCases := []testCase{
		{
			constraint: ">= 1.1.0-beta.1+k0s.1",
			truthTable: map[bool][]string{
				true: {
					"1.1.0+k0s.0",
					"1.1.0-rc.1+k0s.0",
					"1.1.1+k0s.0",
					"1.1.1-rc.1+k0s.0",
				},
				false: {
					"1.1.0-alpha.1+k0s.2",
					"1.0.1+k0s.10",
				},
			},
		},
		{
			constraint: ">= 1.1.0+k0s.1",
			truthTable: map[bool][]string{
				true: {
					"1.1.0+k0s.1",
					"1.1.0+k0s.2",
					"1.1.1+k0s.0",
				},
				false: {
					"1.0.9+k0s.255",
					"1.1.0+k0s.0",
				},
			},
		},
		// simple operator checks
		{
			constraint: "= 1.0.0",
			truthTable: map[bool][]string{
				true:  {"1.0.0"},
				false: {"1.0.1", "0.9.9"},
			},
		},
		{
			constraint: "1.0.0",
			truthTable: map[bool][]string{
				true:  {"1.0.0"},
				false: {"1.0.1", "0.9.9"},
			},
		},
		{
			constraint: "!= 1.0.0",
			truthTable: map[bool][]string{
				true:  {"1.0.1", "0.9.9"},
				false: {"1.0.0"},
			},
		},
		{
			constraint: "> 1.0.0",
			truthTable: map[bool][]string{
				true:  {"1.0.1", "1.1.0"},
				false: {"1.0.0", "0.9.9"},
			},
		},
		{
			constraint: "< 1.0.0",
			truthTable: map[bool][]string{
				true:  {"0.9.9", "0.9.8"},
				false: {"1.0.0", "1.0.1"},
			},
		},
		{
			constraint: ">= 1.0.0",
			truthTable: map[bool][]string{
				true:  {"1.0.0", "1.0.1"},
				false: {"0.9.9"},
			},
		},
		{
			constraint: "<= 1.0.0",
			truthTable: map[bool][]string{
				true:  {"1.0.0", "0.9.9"},
				false: {"1.0.1"},
			},
		},
		// two digit constraints
		{
			constraint: ">= 1.0",
			truthTable: map[bool][]string{
				true:  {"1.0.0", "1.0.1", "1.1.0"},
				false: {"0.9.9", "1.0.1-alpha.1"},
			},
		},
		{
			constraint: ">= 1.0-a",
			truthTable: map[bool][]string{
				true:  {"1.0.0", "1.0.1", "1.0.0-alpha.1"},
				false: {"0.9.9"},
			},
		},
		// loose pre (1.0.1-alpha.1 satisfies >>= 1.0.0 )
		{
			constraint: ">>= 1.0.0",
			truthTable: map[bool][]string{
				true: {
					"1.0.0",
					"1.0.1-rc.1",
					"1.0.1",
				},
				false: {
					"0.9.9-rc.1",
					"1.0.0-alpha.1",
					"1.0.0-rc.1",
				},
			},
		},
		{
			constraint: ">>= 1.0.0-alpha.2",
			truthTable: map[bool][]string{
				true: {
					"1.0.0-alpha.2",
					"1.0.0-alpha.3",
					"1.0.0-rc.1",
					"1.0.0",
					"2.0.1",
				},
				false: {
					"0.9.9-rc.1",
					"0.9.9",
					"1.0.0-alpha.1",
				},
			},
		},
		{
			constraint: "<< 1.0.0-rc.1",
			truthTable: map[bool][]string{
				true: {
					"1.0.0-alpha.1",
					"1.0.0-beta.6",
					"0.9.9",
				},
				false: {
					"1.0.0-rc.1",
					"1.0.1-alpha.1",
					"1.0.0",
				},
			},
		},
		{
			constraint: "<<= 1.0.0-rc.1",
			truthTable: map[bool][]string{
				true: {
					"1.0.0-rc.1",
					"1.0.0-beta.6",
					"0.9.9",
				},
				false: {
					"1.0.1-alpha.1",
					"1.0.0",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.constraint, func(t *testing.T) {
			c, err := version.NewConstraint(tc.constraint)
			NoError(t, err)

			for expected, versions := range tc.truthTable {
				t.Run(fmt.Sprintf("%t", expected), func(t *testing.T) {
					for _, v := range versions {
						t.Run(v, func(t *testing.T) {
							Equal(t, expected, c.Check(version.MustParse(v)))
						})
					}
				})
			}
		})
	}
}

func TestInvalidConstraint(t *testing.T) {
	invalidConstraints := []string{
		"",
		"==",
		">= ",
		"invalid",
		">= abc",
	}

	for _, invalidConstraint := range invalidConstraints {
		_, err := version.NewConstraint(invalidConstraint)
		Error(t, err)
	}
}

func TestCheckString(t *testing.T) {
	c, err := version.NewConstraint(">= 1.0.0")
	NoError(t, err)

	True(t, c.CheckString("1.0.0"))
	False(t, c.CheckString("0.9.9"))
	False(t, c.CheckString("x"))
}

func TestString(t *testing.T) {
	c, err := version.NewConstraint(">= 1.0.0, < 2.0.0")
	NoError(t, err)

	Equal(t, ">= 1.0.0, < 2.0.0", c.String())
}
