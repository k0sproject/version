package version

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCollection(t *testing.T) {
	c, err := NewCollection("1.23.3+k0s.1", "1.23.4+k0s.1")
	assert.NoError(t, err)
	assert.Equal(t, "1.23.3+k0s.1", c[0].String())
	assert.Equal(t, "1.23.4+k0s.1", c[1].String())
	assert.Len(t, c, 2)
	_, err = NewCollection("1.23.3+k0s.1", "1.23.b+k0s.1")
	assert.Error(t, err)
}

func TestSorting(t *testing.T) {
	c, err := NewCollection(
		"1.21.2+k0s.0",
		"1.21.1+k0s.1",
		"0.13.1",
		"1.21.1+k0s.2",
	)
	assert.NoError(t, err)
	sort.Sort(c)
	assert.Equal(t, "0.13.1", c[0].String())
	assert.Equal(t, "1.21.1+k0s.1", c[1].String())
	assert.Equal(t, "1.21.1+k0s.2", c[2].String())
	assert.Equal(t, "1.21.2+k0s.0", c[3].String())
}
