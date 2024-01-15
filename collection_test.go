package version_test

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/k0sproject/version"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCollection(t *testing.T) {
	c, err := version.NewCollection("1.23.3+k0s.1", "1.23.4+k0s.1")
	assert.NoError(t, err)
	assert.Equal(t, "v1.23.3+k0s.1", c[0].String())
	assert.Equal(t, "v1.23.4+k0s.1", c[1].String())
	assert.Len(t, c, 2)
	_, err = version.NewCollection("1.23.3+k0s.1", "1.23.b+k0s.1")
	assert.Error(t, err)
}

func TestSorting(t *testing.T) {
	c, err := version.NewCollection(
		"1.21.2+k0s.0",
		"1.21.2-beta.1+k0s.0",
		"1.21.1+k0s.1",
		"0.13.1",
		"v1.21.1+k0s.2",
	)
	require.NoError(t, err)
	sort.Sort(c)
	assert.Equal(t, "v0.13.1", c[0].String())
	assert.Equal(t, "v1.21.1+k0s.1", c[1].String())
	assert.Equal(t, "v1.21.1+k0s.2", c[2].String())
	assert.Equal(t, "v1.21.2-beta.1+k0s.0", c[3].String())
	assert.Equal(t, "v1.21.2+k0s.0", c[4].String())
	assert.Equal(t, "v1.22.3+k0s.0", c[5].String())
}

func TestCollectionMarshalling(t *testing.T) {
	c, err := version.NewCollection("v1.0.0+k0s.0", "v1.0.1+k0s.0")
	assert.NoError(t, err)

	t.Run("JSON", func(t *testing.T) {
		jsonData, err := json.Marshal(c)
		require.NoError(t, err)
		assert.Equal(t, `["v1.0.0+k0s.0","v1.0.1+k0s.0"]`, string(jsonData))
	})
}

func TestCollectionUnmarshalling(t *testing.T) {
	t.Run("JSON", func(t *testing.T) {
		var c version.Collection
		err := json.Unmarshal([]byte(`["v1.0.0+k0s.1","v1.0.1+k0s.1"]`), &c)
		require.NoError(t, err)
		assert.Equal(t, "v1.0.0+k0s.1", c[0].String())
		assert.Equal(t, "v1.0.1+k0s.1", c[1].String())
	})
}

func TestFailingCollectionUnmarshalling(t *testing.T) {
	t.Run("JSON", func(t *testing.T) {
		var c version.Collection
		err := json.Unmarshal([]byte(`invalid_json`), &c)
		assert.Error(t, err)
		err = json.Unmarshal([]byte(`["invalid_version"]`), &c)
		assert.Error(t, err)
	})
}
