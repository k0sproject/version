# version

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

### `k0s_sort` executable

A command-line interface to the package. Can be used to sort lists of versions or to obtain the latest version number.

```console
Usage: k0s_sort [options] [filename ...]
  -l	only print the latest version
  -o	print the latest version from online
  -s	omit prerelease versions
  -v	print k0s_sort version
```
