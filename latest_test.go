package version_test

import (
	"regexp"
	"testing"

	"github.com/k0sproject/version"
)

func TestLatestByPrerelease(t *testing.T) {
	r, err := version.LatestByPrerelease(false)
	NoError(t, err)
	True(t, regexp.MustCompile(`^v\d+\.\d+\.\d+\+k0s\.\d+$`).MatchString(r.String()))
}
