# version
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fk0sproject%2Fversion.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fk0sproject%2Fversion?ref=badge_shield)


A go-language package for parsing, comparing, sorting and constraint-checking [k0s](https://github.com/k0sproject/k0s) version numbers. The API is modeled after [hashicorp/go-version](https://github.com/hashicorp/go-version). 

K0s versioning follows [semver](https://semver.org/) v2.0 with the exception that there is a special metadata field for the k0s build version like `v1.23.4+k0s.1` which affects precedence while sorting or comparing version numbers.

The library should work fine for performing the same operations on non-k0s version numbers as long as there are maximum of 3 numeric segments (1.2.3), but this is not a priority. There are no dependencies.

## Usage

### Basic comparison

```go
import (
	"fmt"

	"github.com/k0sproject/version"
)

func main() {
	a := version.MustParse("1.23.3+k0s.1")
	b := version.MustParse("1.23.3+k0s.2")
	fmt.Printf("a is greater than b: %t\n", a.GreaterThan(b))
	fmt.Printf("a is less than b: %t\n", a.LessThan(b))
	fmt.Printf("a is equal to b: %t\n", a.Equal(b))
}
```

Outputs:

```text
a is greater than b: false
a is less than b: true
a is equal to b: false
```

### Constraints

```go
import (
	"fmt"

	"github.com/k0sproject/version"
)

func main() {
	v := version.MustParse("1.23.3+k0s.1")
	c := version.MustConstraint("> 1.23")
    fmt.Printf("constraint %s satisfied by %s: %t\n", c, v, c.Check(v))
}
```

Outputs:

```text
constraint > 1.2.3 satisfied by v1.23.3+k0s.1: true
```

### Sorting

```go
import (
	"fmt"
    "sort"

	"github.com/k0sproject/version"
)

func main() {
    versions := []*version.Version{
	    version.MustParse("v1.23.3+k0s.2"),
        version.MustParse("1.23.2+k0s.3"),
        version.MustParse("1.23.3+k0s.1"),
    }

    fmt.Println("Before:")
    for _, v range versions {
        fmt.Println(v)
    }
    sort.Sort(versions)
    fmt.Println("After:")
    for _, v range versions {
        fmt.Println(v)
    }
}
```

Outputs:

```text
Before:
v1.23.3+k0s.2
v1.23.2+k0s.3
v1.23.3+k0s.1
After:
v1.23.2+k0s.3
v1.23.3+k0s.1
v1.23.3+k0s.2
```

### Check online for latest version

```go
import (
	"fmt"
	"github.com/k0sproject/version"
)

func main() {
	latest, err := version.Latest()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Latest k0s version is: %s\n", latest)
}
```

### List released versions

```go
import (
	"context"
	"fmt"

	"github.com/k0sproject/version"
)

func main() {
	ctx := context.Background()
	versions, err := version.All(ctx)
	if err != nil {
		panic(err)
	}
	for _, v := range versions {
		fmt.Println(v)
	}
}
```

The first call hydrates a cache under the OS cache directory (honouring `XDG_CACHE_HOME` when set) and reuses it for subsequent listings.

### `k0s_sort` executable

A command-line interface to the package. It can sort version lists, fetch released tags from GitHub, and compute upgrade paths.

```console
Usage: k0s_sort [options] [filename ...]
  -a	list released versions from GitHub (stable only, honours cache)
  -A	list released versions from GitHub including prereleases
  -d	print version delta instead of upgrade path output
  -l	only print the latest version (works with input or together with -a/-A)
  -s	omit prerelease versions
  -u	require up-to-date online data
  -v	print k0s_sort version

Examples:
  k0s_sort "v1.24.0...stable"
  k0s_sort -d "v1.24.0...v1.26.1"
  k0s_sort -a ">= v1.25.0"
  k0s_sort -u -a -l
  cat versions.txt | k0s_sort -s -l
```


## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fk0sproject%2Fversion.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fk0sproject%2Fversion?ref=badge_large)
