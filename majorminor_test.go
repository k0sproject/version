package version_test

import (
	"testing"

	"github.com/k0sproject/version"
)

func TestMajorMinorString(t *testing.T) {
	mm := version.NewMajorMinor(1, 2)
	if expected, actual := "1.2", mm.String(); expected != actual {
		t.Errorf("Expected %q, got %q", expected, actual)
	}
}

func TestMajorMinor_FromVersion(t *testing.T) {
	v := version.MustParse("v1.23.12-rc.1+k0s.0")
	expected := version.NewMajorMinor(1, 23)
	actual := v.ToMajorMinor()

	if expected != actual {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}

func TestMajorMinor_MatchVersion(t *testing.T) {
	underTest := version.NewMajorMinor(1, 2)

	for _, test := range []struct {
		version  string
		expected bool
	}{
		{"v0.9.1", false},
		{"v1.1.2", false},
		{"v1.2.3", true},
		{"v1.3.4", false},
		{"v2.0.5", false},
	} {
		v := version.MustParse(test.version)
		if actual := underTest.MatchVersion(v); test.expected != actual {
			t.Errorf("Expected (%s).MatchVersion(%q) to be %t, but was %t", underTest, test.version, test.expected, actual)
		}
		if actual := v.Is(underTest); test.expected != actual {
			t.Errorf("Expected %q.Is(%s) to be %t, but was %t", test.version, underTest, test.expected, actual)
		}
	}
}

func TestMajorMinor_Compare(t *testing.T) {
	v12 := version.NewMajorMinor(1, 2)
	v13 := version.NewMajorMinor(1, 3)
	v20 := version.NewMajorMinor(2, 0)

	if actual := v12.Compare(v13); actual != -1 {
		t.Errorf("Expected v12 < v13, got %d", actual)
	}
	if actual := v20.Compare(v12); actual != 1 {
		t.Errorf("Expected v20 > v12, got %d", actual)
	}
	if actual := v12.Compare(v12); actual != 0 {
		t.Errorf("Expected v12 == v12, got %d", actual)
	}
}

func TestMajorMinor_CompareVersion(t *testing.T) {
	v12 := version.NewMajorMinor(1, 2)
	v13 := version.MustParse("1.3.4")
	v09 := version.MustParse("0.9.8")

	if actual := v12.CompareVersion(v13); actual != -1 {
		t.Errorf("Expected v1 < v2, got %d", actual)
	}
	if actual := v12.CompareVersion(v09); actual != 1 {
		t.Errorf("Expected v1 > v3, got %d", actual)
	}
	if actual := v12.CompareVersion(version.MustParse("1.2.2")); actual != 0 {
		t.Errorf("Expected v1 == v1, got %d", actual)
	}
}
