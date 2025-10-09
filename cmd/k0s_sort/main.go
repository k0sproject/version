package main

import (
	"bufio"
	"errors"
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

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		if errWrite := writeFormatted(os.Stderr, "%v\n", err); errWrite != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	commandName := filepath.Base(os.Args[0])
	fs := flag.NewFlagSet("k0s_sort", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		showVersion bool
		latestOnly  bool
		stableOnly  bool
		listStable  bool
		listAll     bool
		pathOnly    bool
	)

	fs.Usage = func() {
		if err := printUsage(fs, commandName, stderr); err != nil {
			reportWriteError(stderr, err)
		}
	}

	fs.BoolVar(&showVersion, "v", false, "print k0s_sort version")
	fs.BoolVar(&latestOnly, "l", false, "print only the latest version")
	fs.BoolVar(&stableOnly, "s", false, "omit prerelease versions")
	fs.BoolVar(&listStable, "a", false, "list released versions from GitHub (stable only)")
	fs.BoolVar(&listAll, "A", false, "list released versions from GitHub including prereleases")
	fs.BoolVar(&pathOnly, "u", false, "show upgrade path between two versions (requires FROM and TO arguments)")
	fs.BoolVar(&pathOnly, "U", false, "show upgrade path between two versions (requires FROM and TO arguments)")
	fs.BoolVar(&pathOnly, "upgrade", false, "show upgrade path between two versions (requires FROM and TO arguments)")

	for _, arg := range args {
		if arg == "--help" || arg == "-?" || arg == "/?" || arg == "help" || arg == "-h" {
			return printUsage(fs, commandName, stdout)
		}
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if showVersion {
		if err := writeLine(stdout, toolversion.Version); err != nil {
			return err
		}
		return nil
	}

	if listStable && listAll {
		return errors.New("flags -a and -A cannot be used together")
	}

	if (listStable || listAll) && pathOnly {
		return errors.New("flags -a/-A and -u cannot be used together")
	}

	if pathOnly {
		return printPath(fs.Args(), stdout, stableOnly)
	}

	if listStable || listAll {
		if fs.NArg() != 0 {
			if listAll {
				return errors.New("no files may be specified with -A")
			}
			return errors.New("no files may be specified with -a")
		}
		if listAll {
			return printAll(stdout, false, latestOnly)
		}
		return printAll(stdout, true, latestOnly)
	}

	return processInput(fs.Args(), stdin, stdout, stableOnly, latestOnly)
}

func printAll(stdout io.Writer, stableOnly, latestOnly bool) error {
	versions, err := version.All()
	if err != nil {
		return fmt.Errorf("fetch versions: %w", err)
	}

	filtered := filterCollection(versions, stableOnly)
	if latestOnly {
		if len(filtered) == 0 {
			return nil
		}
		last := filtered[len(filtered)-1]
		return writeFormatted(stdout, "v%s\n", strings.TrimPrefix(last.String(), "v"))
	}

	for _, v := range filtered {
		if err := writeFormatted(stdout, "v%s\n", strings.TrimPrefix(v.String(), "v")); err != nil {
			return err
		}
	}
	return nil
}

func printPath(args []string, stdout io.Writer, stableOnly bool) error {
	if len(args) != 2 {
		return errors.New("-u requires FROM and TO versions, e.g. -u v1.24.0 v1.26.1")
	}

	from, err := version.NewVersion(args[0])
	if err != nil {
		return fmt.Errorf("parse FROM version: %w", err)
	}

	to, err := version.NewVersion(args[1])
	if err != nil {
		return fmt.Errorf("parse TO version: %w", err)
	}

	path, err := from.UpgradePath(to)
	if err != nil {
		return err
	}

	for _, v := range path {
		if v == nil {
			continue
		}
		if stableOnly && v.IsPrerelease() && !v.Equal(to) {
			continue
		}
		if err := writeFormatted(stdout, "v%s\n", strings.TrimPrefix(v.String(), "v")); err != nil {
			return err
		}
	}
	return nil
}

func processInput(files []string, stdin io.Reader, stdout io.Writer, stableOnly, latestOnly bool) (err error) {
	collection := version.Collection{}

	type namedCloser struct {
		io.Closer
		name string
	}

	var (
		reader  io.Reader
		closers []namedCloser
	)

	defer func() {
		var closeErr error
		for _, c := range closers {
			if c.Closer == nil {
				continue
			}
			if err := c.Close(); err != nil && closeErr == nil {
				closeErr = fmt.Errorf("close %s: %w", c.name, err)
			}
		}
		if err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	if len(files) > 0 {
		readers := make([]io.Reader, 0, len(files))
		for _, fn := range files {
			if fn == "-" {
				readers = append(readers, stdin)
				continue
			}
			file, openErr := os.Open(fn)
			if openErr != nil {
				return fmt.Errorf("open %s: %w", fn, openErr)
			}
			closers = append(closers, namedCloser{Closer: file, name: fn})
			readers = append(readers, file)
		}
		if len(readers) == 0 {
			reader = stdin
		} else {
			reader = io.MultiReader(readers...)
		}
	} else {
		if stdin == nil {
			return errors.New("no input provided")
		}
		if f, ok := stdin.(*os.File); ok {
			info, statErr := f.Stat()
			if statErr != nil {
				return fmt.Errorf("stat stdin: %w", statErr)
			}
			if (info.Mode() & os.ModeCharDevice) != 0 {
				return errors.New("stdin has no data; provide filenames or use -a/-A/-u")
			}
		}
		reader = stdin
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		v, parseErr := version.NewVersion(text)
		if parseErr != nil {
			return fmt.Errorf("parse version %q: %w", text, parseErr)
		}
		if stableOnly && v.IsPrerelease() {
			continue
		}
		collection = append(collection, v)
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return fmt.Errorf("read input: %w", scanErr)
	}

	sort.Sort(collection)

	if len(collection) == 0 {
		return nil
	}

	if latestOnly {
		latest := collection[len(collection)-1]
		return writeFormatted(stdout, "v%s\n", strings.TrimPrefix(latest.String(), "v"))
	}

	for _, v := range collection {
		if err := writeFormatted(stdout, "v%s\n", strings.TrimPrefix(v.String(), "v")); err != nil {
			return err
		}
	}
	return nil
}

func printUsage(fs *flag.FlagSet, commandName string, out io.Writer) error {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Usage: %s [options] [filename ...]\n", commandName))

	prevOutput := fs.Output()
	fs.SetOutput(&builder)
	fs.PrintDefaults()
	fs.SetOutput(prevOutput)

	builder.WriteString("\n")
	builder.WriteString("Examples:\n")
	builder.WriteString(fmt.Sprintf("  %s -u v1.24.0 v1.26.1  # upgrade path between versions\n", commandName))
	builder.WriteString(fmt.Sprintf("  %s -a -l            # latest released stable version\n", commandName))
	builder.WriteString(fmt.Sprintf("  %s -A -l            # latest released version including prereleases\n", commandName))

	_, err := io.WriteString(out, builder.String())
	return err
}

func writeFormatted(w io.Writer, format string, args ...interface{}) error {
	_, err := fmt.Fprintf(w, format, args...)
	return err
}

func writeLine(w io.Writer, text string) error {
	_, err := fmt.Fprintln(w, text)
	return err
}

func reportWriteError(w io.Writer, writeErr error) {
	if writeErr == nil {
		return
	}
	if _, err := fmt.Fprintf(w, "failed to write usage output: %v\n", writeErr); err != nil {
		return
	}
}

func filterCollection(c version.Collection, stableOnly bool) version.Collection {
	if !stableOnly {
		return c
	}
	filtered := make(version.Collection, 0, len(c))
	for _, v := range c {
		if v == nil {
			continue
		}
		if v.IsPrerelease() {
			continue
		}
		filtered = append(filtered, v)
	}
	return filtered
}
