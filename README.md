# version

A go-language package for managing [k0s](https://github.com/k0sproject/k0s) version numbers. It is based on [hashicorp/go-version](https://github.com/hashicorp/go-version) but adds sorting and comparison capabilities for the k0s version numbering scheme which requires additional sorting by the build tag.

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
