package version

import (
	"fmt"
)

// Collection is a type that implements the sort.Interface interface
// so that versions can be sorted.
type Collection []*Version

func NewCollection(versions ...string) (Collection, error) {
	c := make(Collection, len(versions))
	for i, v := range versions {
		nv, err := NewVersion(v)
		if err != nil {
			return Collection{}, fmt.Errorf("invalid version '%s': %w", v, err)
		}
		c[i] = nv
	}
	return c, nil
}

func (c Collection) Len() int {
	return len(c)
}

func (c Collection) Less(i, j int) bool {
	return c[i].Compare(c[j]) < 0
}

func (c Collection) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
