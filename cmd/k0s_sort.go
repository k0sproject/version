package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"

	"github.com/k0sproject/version"
)

func main() {
	stat, err := os.Stdin.Stat()
	if err != nil {
		panic(fmt.Errorf("can't stat stdin: %w", err))
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		panic(fmt.Errorf("can't read stdin"))
	}
	versions := version.Collection{}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		v, err := version.NewVersion(scanner.Text())
		if err != nil {
			panic(fmt.Errorf("failed to parse version: %w", err))
		}
		versions = append(versions, v)
	}

	sort.Sort(versions)

	for _, v := range versions {
		fmt.Println(v)
	}
}
