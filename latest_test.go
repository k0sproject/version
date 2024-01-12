package version_test

import (
	"regexp"
	"testing"

	"github.com/k0sproject/version"

	"github.com/stretchr/testify/assert"
)

func TestLatestByPrerelease(t *testing.T) {
	r, err := version.LatestByPrerelease(false)
	assert.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`^v\d+\.\d+\.\d+\+k0s\.\d+$`), r.String())
}
