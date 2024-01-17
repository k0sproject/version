package version_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/k0sproject/version"
)

func NoError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Received an unexpected error: %v", err)
	}
}

func Error(t *testing.T, err error) {
	if err == nil {
		t.Fatalf("Expected an error, got nil")
	}
}

func Equal(t *testing.T, expected, actual interface{}) {
	if reflect.DeepEqual(expected, actual) {
		return
	}
	t.Errorf("Expected %v, got %v", expected, actual)
}

func True(t *testing.T, actual bool) {
	if actual {
		return
	}
	t.Errorf("Expected true, got false")
}

func False(t *testing.T, actual bool) {
	if !actual {
		return
	}
	t.Errorf("Expected false, got true")
}

func Nil(t *testing.T, actual interface{}) {
	if actual == nil {
		return
	}
	t.Errorf("Expected nil, got %v", actual)
}

func TestNewVersion(t *testing.T) {
	v, err := version.NewVersion("1.23.3+k0s.1")
	NoError(t, err)
	Equal(t, "v1.23.3+k0s.1", v.String())
	_, err = version.NewVersion("v1.23.b+k0s.1")
	Error(t, err)
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
