package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewVersion(t *testing.T) {
	v, err := NewVersion("1.23.3+k0s.1")
	assert.NoError(t, err)
	assert.Equal(t, "1.23.3+k0s.1", v.String())
	v, err = NewVersion("1.23.b+k0s.1")
	assert.Error(t, err)
}

func TestBasicComparison(t *testing.T) {
	a, err := NewVersion("1.23.1+k0s.1")
	assert.NoError(t, err)
	b, err := NewVersion("1.23.2+k0s.1")
	assert.NoError(t, err)
	assert.True(t, b.GreaterThan(a), "version %s should be greater than %s", b, a)
	assert.True(t, a.LessThan(b), "version %s should be less than %s", b, a)
	assert.False(t, b.Equal(a), "version %s should not be equal to %s", b, a)
}

func TestK0sComparison(t *testing.T) {
	a, err := NewVersion("1.23.1+k0s.1")
	assert.NoError(t, err)
	b, err := NewVersion("1.23.1+k0s.2")
	assert.NoError(t, err)
	assert.True(t, b.GreaterThan(a), "version %s should be greater than %s", b, a)
	assert.True(t, a.LessThan(b), "version %s should be less than %s", b, a)
	assert.False(t, b.Equal(a), "version %s should not be equal to %s", b, a)
}
