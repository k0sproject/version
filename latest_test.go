package version

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLatestByPrerelease(t *testing.T) {
	r, err := LatestByPrerelease(false)
	assert.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`^v\d+\.\d+\.\d+\+k0s\.\d+$`), r.String())
}
