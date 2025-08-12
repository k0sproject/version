package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/k0sproject/version"
	toolversion "github.com/k0sproject/version/internal/version"
)

var (
	versionFlag    bool
	latestFlag     bool
	onlineFlag     bool
	stableOnlyFlag bool
)

func online() {
	v, err := version.LatestByPrerelease(!stableOnlyFlag)
	if err != nil {
		println("failed to get latest version:", err.Error())
		os.Exit(1)
	}
	fmt.Println(v.String())
}

func main() {
	flag.Usage = func() {
		exe, _ := os.Executable()
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [filename ...]\n", filepath.Base(exe))
		flag.PrintDefaults()
	}
	flag.BoolVar(&versionFlag, "v", false, "print k0s_sort version")
	flag.BoolVar(&latestFlag, "l", false, "only print the latest version from input")
	flag.BoolVar(&onlineFlag, "o", false, "print the latest version from online")
	flag.BoolVar(&stableOnlyFlag, "s", false, "omit prerelease versions")
	flag.Parse()

	if versionFlag {
		fmt.Println(toolversion.Version)
		return
	}

	if onlineFlag {
		online()
		return
	}

	var input io.Reader
	if flag.NArg() > 0 && flag.Arg(0) != "-" {
		var files []io.Reader
		for _, fn := range flag.Args() {
			file, err := os.Open(fn)
			if err != nil {
				println("can't open file:", err.Error())
				os.Exit(1)
			}
			defer func() {
				if err := file.Close(); err != nil {
					println("can't close file:", err.Error())
				}
			}()
			files = append(files, file)
		}
		input = io.MultiReader(files...)
	} else {
		stat, err := os.Stdin.Stat()
		if err != nil {
			println("can't stat stdin:", err.Error())
			os.Exit(1)
		}
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			println("can't read stdin")
			os.Exit(1)
		}
		input = os.Stdin
	}
	versions := version.Collection{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		v, err := version.NewVersion(scanner.Text())
		if err != nil {
			println("failed to parse version:", err.Error())
			os.Exit(1)
		}
		if v.Prerelease() != "" && stableOnlyFlag {
			continue
		}
		versions = append(versions, v)
	}

	sort.Sort(versions)

	if latestFlag && len(versions) > 0 {
		fmt.Printf("v%s\n", strings.TrimPrefix(versions[len(versions)-1].String(), "v"))
		return
	}

	for _, v := range versions {
		fmt.Printf("v%s\n", strings.TrimPrefix(v.String(), "v"))
	}
}
