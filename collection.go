package version

import (
	"encoding/json"
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

func (c *Collection) marshal() ([]string, error) {
	strSlice := make([]string, len(*c))
	for i, v := range *c {
		s, err := v.MarshalJSON()
		if err != nil {
			return nil, err
		}
		strSlice[i] = string(s)
	}
	return strSlice, nil
}

func (c *Collection) unmarshal(strSlice []string) error {
	coll := make(Collection, len(strSlice))
	for i, s := range strSlice {
		v, err := NewVersion(s)
		if err != nil {
			return err
		}
		coll[i] = v
	}
	*c = coll
	return nil
}

// UnmarshalText implements the json.Marshaler interface.
func (c *Collection) MarshalJSON() ([]byte, error) {
	strSlice, err := c.marshal()
	if err != nil {
		return nil, err
	}
	return json.Marshal(strSlice)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (c *Collection) UnmarshalJSON(data []byte) error {
	var strSlice []string
	if err := json.Unmarshal(data, &strSlice); err != nil {
		return fmt.Errorf("failed to decode JSON input: %w", err)
	}
	return c.unmarshal(strSlice)
}

// MarshalYAML implements the yaml.Marshaler interface.
func (c *Collection) MarshalYAML() (interface{}, error) {
	return c.marshal()
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *Collection) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var strSlice []string
	if err := unmarshal(&strSlice); err != nil {
		return fmt.Errorf("failed to decode YAML input: %w", err)
	}
	return c.unmarshal(strSlice)
}
