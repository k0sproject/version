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
	"unicode"
	"unicode/utf8"

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
		showVersion  bool
		latestOnly   bool
		stableOnly   bool
		listStable   bool
		listAll      bool
		deltaOnly    bool
		requireFresh bool
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
	fs.BoolVar(&deltaOnly, "d", false, "print version delta instead of upgrade path")
	fs.BoolVar(&requireFresh, "u", false, "require up-to-date online data")

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

	parsedArgs := fs.Args()

	if listStable || listAll {
		if len(parsedArgs) > 1 {
			if listAll {
				return errors.New("-A accepts at most one constraint argument")
			}
			return errors.New("-a accepts at most one constraint argument")
		}

		var constraint *version.Constraint
		if len(parsedArgs) == 1 {
			constraintCandidate := parsedArgs[0]
			if !looksLikeConstraint(constraintCandidate) {
				return fmt.Errorf("%q is not a valid constraint argument", constraintCandidate)
			}
			c, err := version.NewConstraint(constraintCandidate)
			if err != nil {
				return fmt.Errorf("parse constraint %q: %w", constraintCandidate, err)
			}
			constraint = &c
		}

		if listAll {
			return printAll(stdout, false, latestOnly, constraint, requireFresh)
		}
		return printAll(stdout, true, latestOnly, constraint, requireFresh)
	}

	if len(parsedArgs) > 0 && strings.Contains(parsedArgs[0], "...") {
		if len(parsedArgs) != 1 {
			return errors.New("upgrade path specification must be provided as a single argument")
		}
		return handleUpgradeSpec(parsedArgs[0], stdout, stableOnly, deltaOnly, requireFresh)
	}

	if deltaOnly {
		return errors.New("-d requires an upgrade path argument containing '...'")
	}

	if len(parsedArgs) > 0 && looksLikeConstraint(parsedArgs[0]) {
		if len(parsedArgs) < 2 {
			return errors.New("constraint checks require at least one version argument")
		}
		c, err := version.NewConstraint(parsedArgs[0])
		if err != nil {
			return fmt.Errorf("parse constraint %q: %w", parsedArgs[0], err)
		}
		for _, candidate := range parsedArgs[1:] {
			v, parseErr := version.NewVersion(candidate)
			if parseErr != nil {
				return fmt.Errorf("parse version %q: %w", candidate, parseErr)
			}
			if !c.Check(v) {
				return fmt.Errorf("version %s does not satisfy %s", v.String(), c.String())
			}
		}
		return nil
	}

	return processInput(parsedArgs, stdin, stdout, stableOnly, latestOnly)
}

func printAll(stdout io.Writer, stableOnly, latestOnly bool, constraint *version.Constraint, requireFresh bool) error {
	versions, err := loadVersions(requireFresh)
	if err != nil {
		return fmt.Errorf("fetch versions: %w", err)
	}

	filtered := filterCollection(versions, stableOnly)
	if constraint != nil {
		filtered = filterByConstraint(filtered, *constraint)
	}

	if len(filtered) == 0 {
		return nil
	}

	if latestOnly {
		last := filtered[len(filtered)-1]
		return writeFormatted(stdout, "v%s\n", strings.TrimPrefix(last.String(), "v"))
	}

	for _, v := range filtered {
		if v == nil {
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
				return errors.New("stdin has no data; provide filenames or use -a/-A")
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
	builder.WriteString(fmt.Sprintf("  %s \"v1.24.0...stable\"    # upgrade path to latest stable\n", commandName))
	builder.WriteString(fmt.Sprintf("  %s -d \"v1.24.0...v1.26.1\" # delta between versions\n", commandName))
	builder.WriteString(fmt.Sprintf("  %s -u -a -l          # latest stable version with fresh data\n", commandName))
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

func filterByConstraint(c version.Collection, constraint version.Constraint) version.Collection {
	filtered := make(version.Collection, 0, len(c))
	for _, v := range c {
		if v == nil {
			continue
		}
		if constraint.Check(v) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func loadVersions(requireFresh bool) (version.Collection, error) {
	if requireFresh {
		return version.Refresh()
	}
	return version.All()
}

func looksLikeConstraint(s string) bool {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return false
	}
	first, _ := utf8DecodeRuneInString(trimmed)
	if first == 'v' || first == 'V' || unicode.IsDigit(first) {
		return false
	}
	return true
}

func utf8DecodeRuneInString(s string) (rune, int) {
	if s == "" {
		return utf8.RuneError, 0
	}
	r, size := utf8.DecodeRuneInString(s)
	return r, size
}

func latestFromCollection(c version.Collection, allowPrerelease bool) (*version.Version, error) {
	for i := len(c) - 1; i >= 0; i-- {
		candidate := c[i]
		if candidate == nil {
			continue
		}
		if !allowPrerelease && candidate.IsPrerelease() {
			continue
		}
		return candidate, nil
	}
	if allowPrerelease {
		return nil, errors.New("no versions available")
	}
	return nil, errors.New("no stable versions available")
}

func resolveFromCollection(c version.Collection, target *version.Version) *version.Version {
	if target == nil {
		return nil
	}
	targetString := target.String()
	for _, candidate := range c {
		if candidate == nil {
			continue
		}
		if candidate.String() == targetString {
			return candidate
		}
	}
	return target
}

func handleUpgradeSpec(spec string, stdout io.Writer, stableOnly, deltaOnly, requireFresh bool) error {
	parts := strings.SplitN(spec, "...", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid upgrade specification %q", spec)
	}

	fromRaw := strings.TrimSpace(parts[0])
	toRaw := strings.TrimSpace(parts[1])
	if fromRaw == "" {
		return errors.New("upgrade specification requires a starting version")
	}

	fromVersion, err := version.NewVersion(fromRaw)
	if err != nil {
		return fmt.Errorf("parse FROM version: %w", err)
	}

	versions, err := loadVersions(requireFresh)
	if err != nil {
		return fmt.Errorf("load versions: %w", err)
	}

	var target *version.Version
	switch {
	case toRaw == "":
		if fromVersion.IsPrerelease() {
			target, err = latestFromCollection(versions, true)
		} else {
			target, err = latestFromCollection(versions, false)
		}
	case strings.EqualFold(toRaw, "stable"):
		target, err = latestFromCollection(versions, false)
	case strings.EqualFold(toRaw, "latest"):
		target, err = latestFromCollection(versions, true)
	default:
		target, err = version.NewVersion(toRaw)
		if err != nil {
			return fmt.Errorf("parse TO version: %w", err)
		}
	}
	if err != nil {
		return fmt.Errorf("determine target version: %w", err)
	}

	if target == nil {
		return errors.New("no target version could be determined")
	}

	target = resolveFromCollection(versions, target)

	if deltaOnly {
		delta := version.NewDelta(fromVersion, target)
		return writeLine(stdout, delta.String())
	}

	path, err := fromVersion.UpgradePath(target)
	if err != nil {
		return err
	}

	for _, v := range path {
		if v == nil {
			continue
		}
		if stableOnly && v.IsPrerelease() && !v.Equal(target) {
			continue
		}
		if err := writeFormatted(stdout, "v%s\n", strings.TrimPrefix(v.String(), "v")); err != nil {
			return err
		}
	}

	return nil
}
