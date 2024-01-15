package version_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/k0sproject/version"

	"github.com/stretchr/testify/assert"
)

func TestNewVersion(t *testing.T) {
	v, err := version.NewVersion("1.23.3+k0s.1")
	assert.NoError(t, err)
	assert.Equal(t, "v1.23.3+k0s.1", v.String())
	_, err = version.NewVersion("v1.23.b+k0s.1")
	assert.Error(t, err)
}

func TestBasicComparison(t *testing.T) {
	a, err := version.NewVersion("1.23.1+k0s.1")
	assert.NoError(t, err)
	b, err := version.NewVersion("1.23.2+k0s.1")
	assert.NoError(t, err)
	assert.True(t, b.GreaterThan(a), "version %s should be greater than %s", b, a)
	assert.True(t, a.LessThan(b), "version %s should be less than %s", b, a)
	assert.False(t, b.Equal(a), "version %s should not be equal to %s", b, a)
}

func TestK0sComparison(t *testing.T) {
	a, err := version.NewVersion("1.23.1+k0s.1")
	assert.NoError(t, err)
	b, err := version.NewVersion("1.23.1+k0s.2")
	assert.NoError(t, err)
	assert.True(t, b.GreaterThan(a), "version %s should be greater than %s", b, a)
	assert.False(t, a.GreaterThan(a), "version %s should not be greater than %s", a, a)
	assert.True(t, a.LessThan(b), "version %s should be less than %s", b, a)
	assert.False(t, a.LessThan(a), "version %s should not be less than %s", a, a)
	assert.False(t, b.Equal(a), "version %s should not be equal to %s", b, a)
}

func TestSatisfies(t *testing.T) {
	v, err := version.NewVersion("1.23.1+k0s.1")
	assert.NoError(t, err)
	assert.True(t, v.Satisfies(version.MustConstraint(">=1.23.1")))
	assert.True(t, v.Satisfies(version.MustConstraint(">=1.23.1+k0s.0")))
	assert.True(t, v.Satisfies(version.MustConstraint(">=1.23.1+k0s.1")))
	assert.True(t, v.Satisfies(version.MustConstraint("=1.23.1+k0s.1")))
	assert.True(t, v.Satisfies(version.MustConstraint("<1.23.1+k0s.2")))
	assert.False(t, v.Satisfies(version.MustConstraint(">=1.23.1+k0s.2")))
	assert.False(t, v.Satisfies(version.MustConstraint(">=1.23.2")))
	assert.False(t, v.Satisfies(version.MustConstraint(">1.23.1+k0s.1")))
	assert.False(t, v.Satisfies(version.MustConstraint("<1.23.1+k0s.1")))
}

func TestURLs(t *testing.T) {
	a, err := version.NewVersion("1.23.3+k0s.1")
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/k0sproject/k0s/releases/tag/v1.23.3%2Bk0s.1", a.URL())
	assert.Equal(t, "https://github.com/k0sproject/k0s/releases/download/v1.23.3%2Bk0s.1/k0s-v1.23.3+k0s.1-amd64.exe", a.DownloadURL("windows", "amd64"))
	assert.Equal(t, "https://github.com/k0sproject/k0s/releases/download/v1.23.3%2Bk0s.1/k0s-v1.23.3+k0s.1-arm64", a.DownloadURL("linux", "arm64"))
	assert.Equal(t, "https://docs.k0sproject.io/v1.23.3+k0s.1/", a.DocsURL())
}

func TestMarshalling(t *testing.T) {
	v, err := version.NewVersion("v1.0.0+k0s.0")
	assert.NoError(t, err)

	t.Run("JSON", func(t *testing.T) {
		jsonData, err := json.Marshal(v)
		assert.NoError(t, err)
		assert.Equal(t, `"v1.0.0+k0s.0"`, string(jsonData))
	})

	t.Run("YAML", func(t *testing.T) {
		yamlData, err := v.MarshalYAML()
		assert.NoError(t, err)
		assert.Equal(t, "v1.0.0+k0s.0", yamlData)
	})

	t.Run("JSON with nil", func(t *testing.T) {
		jsonData, err := json.Marshal(nil)
		assert.NoError(t, err)
		assert.Equal(t, `null`, string(jsonData))
	})

	t.Run("YAML", func(t *testing.T) {
		var nilVersion *version.Version
		yamlData, err := nilVersion.MarshalYAML()
		assert.NoError(t, err)
		assert.Nil(t, yamlData)
	})
}

func TestUnmarshalling(t *testing.T) {
	t.Run("JSON", func(t *testing.T) {
		v := &version.Version{}
		err := json.Unmarshal([]byte(`"v1.0.0+k0s.1"`), v)
		assert.NoError(t, err)
		assert.Equal(t, "v1.0.0+k0s.1", v.String())
	})

	t.Run("YAML", func(t *testing.T) {
		v := &version.Version{}
		err := v.UnmarshalYAML(func(i interface{}) error {
			*(i.(*string)) = "v1.0.0+k0s.1"
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, "v1.0.0+k0s.1", v.String())
	})

	t.Run("JSON with null", func(t *testing.T) {
		v := &version.Version{}
		err := json.Unmarshal([]byte(`null`), v)
		assert.NoError(t, err)
		assert.True(t, v.IsZero())
	})

	t.Run("YAML with empty", func(t *testing.T) {
		v := &version.Version{}
		err := v.UnmarshalYAML(func(i interface{}) error {
			*(i.(*string)) = ""
			return nil
		})
		assert.NoError(t, err)
		assert.Nil(t, nil, v)
	})
}

func TestFailingUnmarshalling(t *testing.T) {
	t.Run("JSON", func(t *testing.T) {
		var v version.Version
		err := json.Unmarshal([]byte(`invalid_json`), &v)
		assert.Error(t, err)
		err = json.Unmarshal([]byte(`"invalid_version"`), &v)
		assert.Error(t, err)
	})

	t.Run("YAML", func(t *testing.T) {
		var v = &version.Version{}
		err := v.UnmarshalYAML(func(i interface{}) error {
			return errors.New("forced error")
		})
		assert.Error(t, err)
		err = v.UnmarshalYAML(func(i interface{}) error {
			*(i.(*string)) = "invalid_version"
			return nil
		})
		assert.Error(t, err)
	})
}
