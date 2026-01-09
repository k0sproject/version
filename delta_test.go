package version_test

import (
	"fmt"
	"testing"

	"github.com/k0sproject/version"
)

func TestDelta(t *testing.T) {
	tests := []struct {
		a, b   string
		expect string
	}{
		{"v1.0.0", "v1.0.1", "a consecutive patch upgrade to v1.0.1"},
		{"v1.0.1", "v1.0.3", "a non-consecutive patch upgrade to v1.0.3"},
		{"v1.0.0", "v1.1.0", "a consecutive minor upgrade from v1.0 to v1.1"},
		{"v1.0.0", "v2.0.0", "a consecutive major upgrade from v1 to v2"},
		{"v1.0.1", "v1.0.0", "v1.0.0 is a downgrade from v1.0.1"},
		{"v1.0.0-alpha", "v1.0.0", "an upgrade from a v1.0.0 pre-release to stable"},
		{"v1.0.0-alpha.1", "v1.0.0-alpha.2", "an upgrade between pre-release versions of v1.0.0"},
		{"v1.0.0+build1", "v1.0.0+build2", "build metadata changes from \"build1\" to \"build2\""},
		{"v1.0.0", "v1.0.0", "no change"},
		{"v1.0.0-rc.1+k0s.1", "v1.0.0-rc.1+k0s.1", "no change"},
		{"v1.1.1", "v2.1.0", "a non-consecutive major upgrade from v1 to v2"},
		{"v1.1.1", "v1.2.0", "a consecutive minor upgrade from v1.1 to v1.2"},
		{"v1.1.1+k0s.0", "v1.1.1+k0s.2", "a non-consecutive k0s upgrade to k0s build 2"},
		{"v1.1.1+k0s.0", "v1.1.1+k0s.1", "a consecutive k0s upgrade to k0s build 1"},
		{"v1.1.1+k0s.0", "v1.3", "a non-consecutive minor upgrade from v1.1 to v1.3"},
		{"v1.1.1+k0s.0", "v2", "a consecutive major upgrade from v1 to v2"},
	}

	for _, test := range tests {
		t.Run("delta from "+test.a+" to "+test.b, func(t *testing.T) {
			a, err := version.NewVersion(test.a)
			NoError(t, err)
			b, err := version.NewVersion(test.b)
			NoError(t, err)
			delta := version.NewDelta(a, b)
			if result := delta.String(); result != test.expect {
				t.Errorf("expected: %q, got: %q", test.expect, result)
			}
		})
	}
}

func ExampleDelta() {
	a, _ := version.NewVersion("v1.0.0")
	b, _ := version.NewVersion("v1.2.1")
	delta := version.NewDelta(a, b)
	_, _ = fmt.Printf("patch upgrade: %t\n", delta.PatchUpgrade)
	_, _ = fmt.Println(delta.String())
	// Output:
	// patch upgrade: false
	// a non-consecutive minor upgrade from v1.0 to v1.2
}
