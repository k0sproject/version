package version

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLatestReleaseByPrerelease(t *testing.T) {
	r, err := LatestReleaseByPrerelease(false)
	assert.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`\d+\.\d+\.\d+\+k0s\.\d+`), r)
}
