package version_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/k0sproject/version"
)

func NoError(t *testing.T, err error) {
	t.Helper() // these make the log messages display the correct line number
	if err != nil {
		t.Fatalf("Received an unexpected error: %v", err)
	}
}

func Error(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("Expected an error, got nil")
	}
}

func Equal(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if reflect.DeepEqual(expected, actual) {
		return
	}
	t.Errorf("Expected %v, got %v", expected, actual)
}

func True(t *testing.T, actual bool) {
	t.Helper()
	if actual {
		return
	}
	t.Errorf("Expected true, got false")
}

func False(t *testing.T, actual bool) {
	t.Helper()
	if !actual {
		return
	}
	t.Errorf("Expected false, got true")
}

func Nil(t *testing.T, actual interface{}) {
	t.Helper()
	if actual == nil {
		return
	}
	t.Errorf("Expected nil, got %v", actual)
}

func TestNewVersion(t *testing.T) {
	v, err := version.NewVersion("1.23.3+k0s.1")
	NoError(t, err)
	Equal(t, "v1.23.3+k0s.1", v.String())
	Equal(t, "v1.23.3", v.Base())
	_, err = version.NewVersion("v1.23.b+k0s.1")
	Error(t, err)
}

func TestWithK0s(t *testing.T) {
	v, err := version.NewVersion("1.23.3+k0s.1")
	NoError(t, err)
	True(t, v.IsK0s())
	k0s, ok := v.K0s()
	Equal(t, 1, k0s)
	True(t, ok)
	v2 := v.WithK0s(2)
	NoError(t, err)
	Equal(t, "v1.23.3+k0s.2", v2.String())
	k0s, ok = v2.K0s()
	Equal(t, 2, k0s)
	True(t, ok)
	// ensure original didnt change
	k0s, ok = v.K0s()
	True(t, ok)
	Equal(t, 1, k0s)

	v, err = version.NewVersion("1.23.3")
	NoError(t, err)
	False(t, v.IsK0s())
	v2 = v.WithK0s(2)
	NoError(t, err)
	Equal(t, "v1.23.3+k0s.2", v2.String())
	// ensure original didnt change
	False(t, v.IsK0s())
	_, ok = v.K0s()
	False(t, ok)
}

func TestBasicComparison(t *testing.T) {
	a, err := version.NewVersion("1.23.1+k0s.1")
	NoError(t, err)
	b, err := version.NewVersion("1.23.2+k0s.1")
	NoError(t, err)
	True(t, b.GreaterThan(a))
	True(t, a.LessThan(b))
	False(t, b.Equal(a))
}

func TestK0sComparison(t *testing.T) {
	a, err := version.NewVersion("1.23.1+k0s.1")
	NoError(t, err)
	b, err := version.NewVersion("1.23.1+k0s.2")
	NoError(t, err)
	True(t, b.GreaterThan(a))
	False(t, a.GreaterThan(a))
	True(t, a.LessThan(b))
	False(t, a.LessThan(a))
	False(t, b.Equal(a))
}

func TestVersion_TotalOrder(t *testing.T) {
	// Store the ordering in a file for manual inspection.
	orderingFilePath := filepath.Join("testdata", "version-ordering.txt")
	data, err := os.ReadFile(orderingFilePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}

		data = generateVersionOrdering()
		if err := os.WriteFile(orderingFilePath, data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	var totalOrder [][]*version.Version
	lines := bufio.NewScanner(bytes.NewReader(data))
	for lines.Scan() {
		var lineNo uint
		var eq []*version.Version
		for _, v := range strings.Split(lines.Text(), " ") {
			lineNo++
			parsed, err := version.NewVersion(v)
			if err != nil {
				t.Fatalf("Failed to parse %q on line %d: %v", parsed, lineNo, err)
			}
			eq = append(eq, parsed)
		}

		totalOrder = append(totalOrder, eq)
	}

	for _, test := range []struct {
		name     string
		op       func(l, r *version.Version) bool
		expected func(l, r int) bool
	}{{
		"Equal",
		func(l, r *version.Version) bool { return l.Equal(r) },
		func(l, r int) bool { return l == r },
	}, {
		"LessThan",
		func(l, r *version.Version) bool { return l.LessThan(r) },
		func(l, r int) bool { return l < r },
	}, {
		"GreaterThan",
		func(l, r *version.Version) bool { return l.GreaterThan(r) },
		func(l, r int) bool { return l > r },
	}, {
		"LessThanOrEqual",
		func(l, r *version.Version) bool { return l.LessThanOrEqual(r) },
		func(l, r int) bool { return l <= r },
	}, {
		"GreaterThanOrEqual",
		func(l, r *version.Version) bool { return l.GreaterThanOrEqual(r) },
		func(l, r int) bool { return l >= r },
	}} {
		t.Run(test.name, func(t *testing.T) {
			for rIdx, r := range totalOrder {
				for lIdx, l := range totalOrder {
					for _, r := range r {
						for _, l := range l {
							if expected, actual := test.expected(lIdx, rIdx), test.op(l, r); expected != actual {
								t.Errorf("Expected %q.%s(%q) to be %t, but was %t", l, test.name, r, expected, actual)
							}
						}
					}
				}
			}
		})
	}
}

func generateVersionOrdering() []byte {
	var buf bytes.Buffer

	maj := []uint{0, 1}
	min := []uint{0, 1}
	pat := []uint{0, 1}
	pre := []string{"-0", "-z", ""}
	meta := []string{"", "+0", "+z"}
	k0s := []string{"", "+k0s.0", "+k0s.1"}

	for _, maj := range maj {
		for _, min := range min {
			for _, pat := range pat {
				for _, pre := range pre {
					for _, k0s := range k0s {
						// All the metas are equal to each other.
						meta := meta

						// Except if it's a "k0s meta". Those are ordered.
						if k0s != "" {
							meta = []string{k0s}
						}

						// Write out all the equal version strings in a single line, space separated.
						for i, meta := range meta {
							if i > 0 {
								buf.WriteByte(' ')
							}
							// Trailing zero minor and patch versions are equal, as well.
							if pat == 0 {
								if min == 0 {
									fmt.Fprintf(&buf, "%d%s%s ", maj, pre, meta)
								}
								fmt.Fprintf(&buf, "%d.%d%s%s ", maj, min, pre, meta)
							}
							fmt.Fprintf(&buf, "%d.%d.%d%s%s", maj, min, pat, pre, meta)
						}
						buf.WriteByte('\n')
					}
				}
			}
		}
	}

	return buf.Bytes()
}

func TestSatisfies(t *testing.T) {
	v, err := version.NewVersion("1.23.1+k0s.1")
	NoError(t, err)
	True(t, v.Satisfies(version.MustConstraint(">=1.23.1")))
	True(t, v.Satisfies(version.MustConstraint(">=1.23.1+k0s.0")))
	True(t, v.Satisfies(version.MustConstraint(">=1.23.1+k0s.1")))
	True(t, v.Satisfies(version.MustConstraint("=1.23.1+k0s.1")))
	True(t, v.Satisfies(version.MustConstraint("<1.23.1+k0s.2")))
	False(t, v.Satisfies(version.MustConstraint(">=1.23.1+k0s.2")))
	False(t, v.Satisfies(version.MustConstraint(">=1.23.2")))
	False(t, v.Satisfies(version.MustConstraint(">1.23.1+k0s.1")))
	False(t, v.Satisfies(version.MustConstraint("<1.23.1+k0s.1")))
}

func TestURLs(t *testing.T) {
	a, err := version.NewVersion("1.23.3+k0s.1")
	NoError(t, err)
	Equal(t, "https://github.com/k0sproject/k0s/releases/tag/v1.23.3%2Bk0s.1", a.URL())
	Equal(t, "https://github.com/k0sproject/k0s/releases/download/v1.23.3%2Bk0s.1/k0s-v1.23.3+k0s.1-amd64.exe", a.DownloadURL("windows", "amd64"))
	Equal(t, "https://github.com/k0sproject/k0s/releases/download/v1.23.3%2Bk0s.1/k0s-v1.23.3+k0s.1-arm64", a.DownloadURL("linux", "arm64"))
	Equal(t, "https://docs.k0sproject.io/v1.23.3+k0s.1/", a.DocsURL())
}

func TestMarshalling(t *testing.T) {
	v, err := version.NewVersion("v1.0.0+k0s.0")
	NoError(t, err)

	t.Run("JSON", func(t *testing.T) {
		jsonData, err := json.Marshal(v)
		NoError(t, err)
		Equal(t, `"v1.0.0+k0s.0"`, string(jsonData))
	})

	t.Run("YAML", func(t *testing.T) {
		yamlData, err := v.MarshalYAML()
		NoError(t, err)
		Equal(t, "v1.0.0+k0s.0", yamlData)
	})

	t.Run("JSON with nil", func(t *testing.T) {
		jsonData, err := json.Marshal(nil)
		NoError(t, err)
		Equal(t, `null`, string(jsonData))
	})

	t.Run("YAML", func(t *testing.T) {
		var nilVersion *version.Version
		yamlData, err := nilVersion.MarshalYAML()
		NoError(t, err)
		Nil(t, yamlData)
	})
}

func TestUnmarshalling(t *testing.T) {
	t.Run("JSON", func(t *testing.T) {
		v := &version.Version{}
		err := json.Unmarshal([]byte(`"v1.0.0+k0s.1"`), v)
		NoError(t, err)
		Equal(t, "v1.0.0+k0s.1", v.String())
	})

	t.Run("YAML", func(t *testing.T) {
		v := &version.Version{}
		err := v.UnmarshalYAML(func(i interface{}) error {
			*(i.(*string)) = "v1.0.0+k0s.1"
			return nil
		})
		NoError(t, err)
		Equal(t, "v1.0.0+k0s.1", v.String())
	})

	t.Run("JSON with null", func(t *testing.T) {
		v := &version.Version{}
		err := json.Unmarshal([]byte(`null`), v)
		NoError(t, err)
		True(t, v.IsZero())
	})

	t.Run("YAML with empty", func(t *testing.T) {
		v := &version.Version{}
		err := v.UnmarshalYAML(func(i interface{}) error {
			*(i.(*string)) = ""
			return nil
		})
		t.Logf("what the shit")
		NoError(t, err)
		True(t, v.IsZero())
	})
}

func TestFailingUnmarshalling(t *testing.T) {
	t.Run("JSON", func(t *testing.T) {
		var v version.Version
		err := json.Unmarshal([]byte(`invalid_json`), &v)
		Error(t, err)
		err = json.Unmarshal([]byte(`"invalid_version"`), &v)
		Error(t, err)
	})

	t.Run("YAML", func(t *testing.T) {
		var v = &version.Version{}
		err := v.UnmarshalYAML(func(i interface{}) error {
			return errors.New("forced error")
		})
		Error(t, err)
		err = v.UnmarshalYAML(func(i interface{}) error {
			*(i.(*string)) = "invalid_version"
			return nil
		})
		Error(t, err)
	})
}
