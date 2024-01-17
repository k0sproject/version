package version_test

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/k0sproject/version"
)

func TestNewCollection(t *testing.T) {
	c, err := version.NewCollection("1.23.3+k0s.1", "1.23.4+k0s.1")
	NoError(t, err)
	Equal(t, "v1.23.3+k0s.1", c[0].String())
	Equal(t, "v1.23.4+k0s.1", c[1].String())
	Equal(t, len(c), 2)
	_, err = version.NewCollection("1.23.3+k0s.1", "1.23.b+k0s.1")
	Error(t, err)
}

func TestSorting(t *testing.T) {
	c, err := version.NewCollection(
		"1.21.2+k0s.0",
		"1.21.2-beta.1+k0s.0",
		"1.21.1+k0s.1",
		"0.13.1",
		"v1.21.1+k0s.2",
	)
	NoError(t, err)
	sort.Sort(c)
	Equal(t, "v0.13.1", c[0].String())
	Equal(t, "v1.21.1+k0s.1", c[1].String())
	Equal(t, "v1.21.1+k0s.2", c[2].String())
	Equal(t, "v1.21.2-beta.1+k0s.0", c[3].String())
	Equal(t, "v1.21.2+k0s.0", c[4].String())
}

func TestCollectionMarshalling(t *testing.T) {
	c, err := version.NewCollection("v1.0.0+k0s.0", "v1.0.1+k0s.0")
	NoError(t, err)

	t.Run("JSON", func(t *testing.T) {
		jsonData, err := json.Marshal(c)
		NoError(t, err)
		Equal(t, `["v1.0.0+k0s.0","v1.0.1+k0s.0"]`, string(jsonData))
	})
}

func TestCollectionUnmarshalling(t *testing.T) {
	t.Run("JSON", func(t *testing.T) {
		var c version.Collection
		err := json.Unmarshal([]byte(`["v1.0.0+k0s.1","v1.0.1+k0s.1"]`), &c)
		NoError(t, err)
		Equal(t, "v1.0.0+k0s.1", c[0].String())
		Equal(t, "v1.0.1+k0s.1", c[1].String())
	})
}

func TestFailingCollectionUnmarshalling(t *testing.T) {
	t.Run("JSON", func(t *testing.T) {
		var c version.Collection
		err := json.Unmarshal([]byte(`invalid_json`), &c)
		Error(t, err)
		err = json.Unmarshal([]byte(`["invalid_version"]`), &c)
		Error(t, err)
	})
}
