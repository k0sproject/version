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

#### Operators

Single `<` and `>` do not match prereleases unless the constraint itself is also a prerelease. Use `<<` and `>>` to allow prereleases to satisfy a stable constraint.

| OP  | Description                                    |
| --- | ---------------------------------------------- |
| <   | Less than                                      |
| <<  | Less than, allowing prereleases                |
| <=  | Less than or equal to                          |
| <== | Less than or equal to, allowing prereleases    |
| >   | Greater than                                   |
| >>  | Greater than, allowing prereleases             |
| >=  | Greater than or equal to                       |
| >== | Greater than or equal to, allowing prereleases |
| ==  | Equal to                                       |
| !=  | Not equal to                                   |

Examples:

- `< 1.0.0` 
  - Matches `0.9.9`.
  - Does not match `1.0.0`.
  - Does not match `0.9.9-rc.2` or `1.0.0-alpha.1` because the constraint is stable.
- `< 1.0.0-rc.1`
  - Matches `0.9.9` and `1.0.0-alpha.1` because the constraint defines a pre-release.
  - Does not match `1.0.0` or `1.0.0-rc.2`.
- `<< 1.0.0`
  - Matches `0.9.9`.
  - Matches `1.0.0-alpha.1` because the pre-release inclusive `<<` operator was used.
  - Does not match `1.0.0`.

Constraints can be combined using a comma (`,`) to form multiple ranges. All ranges must be satisfied for a version to match the constraint. This allows precise control over acceptable version ranges.

- `>= 1.0.0, < 2.0.0`:
  - Matches versions from `1.0.0` (inclusive) up to `2.0.0` (exclusive).
  - Does not match any prereleases.

- `>>= 1.0.0, << 2.0.0`:
  - Matches versions from `1.0.0` (inclusive) up to `2.0.0` (exclusive), including pre-releases of `2.0.0`.
  - Matches `2.0.0-rc.1` but not `2.0.0`.
  - Matches `1.0.1-alpha.1` but not `1.0.0-rc.1`.
- `>> 1.0.0, < 2.0.0`
  - Matches `1.1.0`.
  - Does not match `1.1.0-rc.1` because the second part will not match any pre-releases, any pre-releases matched by the first part get rejected.


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
