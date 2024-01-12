# version

A go-language package for parsing, comparing and sorting [k0s](https://github.com/k0sproject/k0s) version numbers. The API is modeled after [hashicorp/go-version](https://github.com/hashicorp/go-version). 

K0s versioning follows [semver](https://semver.org/) v2.0 with the exception that there is a special metadata field for the k0s build version like `v1.23.4+k0s.1` which affects precedence while sorting or comparing version numbers.

The library should work fine for performing the same operations on non-k0s version numbers as long as there are maximum of 3 numeric segments (1.2.3), but this is not a priority.

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
