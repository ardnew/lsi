package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

const (
	command = "lsi"

	mountPointSymbol = "@"

	// shortRevisionLength is the number of characters to display from a git
	// commit hash. This matches the convention used by git log --oneline and
	// is typically sufficient to uniquely identify a commit in most repositories.
	shortRevisionLength = 7

	// defaultTimeout is the default duration before canceling path traversal.
	defaultTimeout = 0
)

// options holds all command-line flag values.
type options struct {
	version  bool
	timeout  time.Duration
	noFollow bool
	long     bool
	mode     bool
	user     bool
	group    bool
	size     bool
	inode    bool
	mount    bool
}

// widths tracks the maximum width needed for each column.
type widths struct {
	mode, user, group, size, inode int
}

func usage(w io.Writer) {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(w, "usage:\n")
		fmt.Fprintf(w, "  %s [flags] [--] [PATH ...]\n", command)
	} else {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		exe = filepath.Base(exe)
		fmt.Fprintf(w, "usage:\n")
		fmt.Fprintf(w, "  %s [flags] [--] [PATH ...]\n", exe)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "flags:")
	fmt.Fprintln(w, "  -v  Display version information.")
	fmt.Fprintln(w, "  -t  Timeout duration (e.g., 30s, 5m). Default: unlimited.")
	fmt.Fprintln(w, "  -n  Do not follow symlinks.")
	fmt.Fprintln(w, "  -l  Output using long format (-p -u -g -s -m).")
	fmt.Fprintln(w, "  -p  Output file type and permissions.")
	fmt.Fprintln(w, "  -u  Output file owner.")
	fmt.Fprintln(w, "  -g  Output file group.")
	fmt.Fprintln(w, "  -s  Output file size (bytes).")
	fmt.Fprintln(w, "  -i  Output file inode.")
	fmt.Fprintln(w, "  -m  Output mount point symbols ("+mountPointSymbol+").")
}

// atob parses a string as a boolean value.
func atob(s string) bool {
	b, err := strconv.ParseBool(s)
	return err == nil && b
}

// getVersion returns the version string for the lsi binary.
// It attempts to extract version information from VCS (git) tags via build info.
// If no version is available, it returns "(devel)".
func getVersion() string {
	const unversioned = "(devel)"

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return unversioned
	}

	// Check if Main.Version has the version (from git tag)
	if info.Main.Version != "" && info.Main.Version != unversioned {
		return info.Main.Version
	}

	// Try to build version from VCS information
	var revision, modified string
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if len(setting.Value) >= shortRevisionLength {
				revision = setting.Value[:shortRevisionLength]
			} else {
				revision = setting.Value
			}
		case "vcs.modified":
			if atob(setting.Value) {
				modified = "-modified"
			}
		}
	}

	if revision != "" {
		return revision + modified
	}

	return unversioned
}

// contextError creates an error describing why context was canceled.
func contextError(ctx context.Context, start time.Time) error {
	if ctx.Err() == context.DeadlineExceeded {
		elapsed := time.Since(start)
		return fmt.Errorf("timeout after %v", elapsed.Round(time.Millisecond))
	}
	return ctx.Err()
}

// printError formats and prints an entry error.
func printError(w io.Writer, e entry) {
	switch err := e.Err.(type) {
	case *os.PathError:
		fmt.Fprintf(w, " * %s (%s): %s\n", e.Name, err.Path, err.Err)
	default:
		fmt.Fprintf(w, " * %s: %s\n", e.Name, err)
	}
}

func main() {
	if err := run(context.Background(), os.Stdout, os.Stderr, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", command, err)
		os.Exit(1)
	}
}

// run executes the main program logic.
func run(ctx context.Context, out, errOut io.Writer, args []string) error {
	var opts options

	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.Usage = func() { usage(errOut) }

	fs.BoolVar(&opts.version, "v", false, "")
	fs.DurationVar(&opts.timeout, "t", defaultTimeout, "")
	fs.BoolVar(&opts.noFollow, "n", false, "")
	fs.BoolVar(&opts.long, "l", false, "")
	fs.BoolVar(&opts.mode, "p", false, "")
	fs.BoolVar(&opts.user, "u", false, "")
	fs.BoolVar(&opts.group, "g", false, "")
	fs.BoolVar(&opts.size, "s", false, "")
	fs.BoolVar(&opts.inode, "i", false, "")
	fs.BoolVar(&opts.mount, "m", false, "")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// If version requested, print it and return.
	if opts.version {
		fmt.Fprintf(out, "%s %s\n", command, getVersion())
		return nil
	}

	// Determine the file paths to analyze.
	paths := fs.Args()
	if len(paths) == 0 {
		// If no paths were given, use PWD.
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}
		paths = []string{wd}
	}

	// Configure the meta-flags.
	if opts.long {
		opts.mode, opts.user, opts.group, opts.size, opts.mount = true, true, true, true, true
	}

	// Set up timeout if specified.
	if opts.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.timeout)
		defer cancel()
	}

	// Process each path.
	for i, p := range paths {
		// If more than one path provided, print a header for the current path.
		if len(paths) > 1 {
			fp := filepath.Clean(p)
			fmt.Fprintf(out, "-- %s\n", fp)
		}

		if err := processPath(ctx, out, p, opts); err != nil {
			return err
		}

		if len(paths) > 1 && i+1 < len(paths) {
			fmt.Fprintln(out)
		}
	}

	return nil
}

// processPath walks a single path and prints its entries.
func processPath(ctx context.Context, out io.Writer, path string, opts options) error {
	start := time.Now()

	entries, err := collectEntries(ctx, path, opts)
	if err != nil {
		if ctx.Err() != nil {
			return contextError(ctx, start)
		}
		return err
	}

	w := calculateWidths(entries)
	printEntries(out, entries, opts, w)
	return nil
}

// collectEntries performs the path walk and collects all entries.
func collectEntries(ctx context.Context, path string, opts options) ([]entry, error) {
	var entries []entry

	err := walk(ctx, path, func(ctx context.Context, e entry) (bool, error) {
		entries = append(entries, e)
		// Stop building buffer on the first error encountered.
		if e.Err != nil {
			return false, e.Err
		}
		// Check for context cancellation.
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		return !opts.noFollow, nil
	})

	return entries, err
}

// calculateWidths determines the maximum width for each column.
func calculateWidths(entries []entry) widths {
	var w widths
	for _, e := range entries {
		w.mode = max(w.mode, len(e.Mode))
		w.user = max(w.user, len(e.User))
		w.group = max(w.group, len(e.Group))
		w.size = max(w.size, len(strconv.FormatInt(e.Size, 10)))
		w.inode = max(w.inode, len(strconv.FormatUint(e.Inode, 10)))
	}
	return w
}

// printEntries prints all entries with appropriate formatting.
func printEntries(w io.Writer, entries []entry, opts options, widths widths) {
	for _, e := range entries {
		if e.Err != nil {
			printError(w, e)
		} else {
			e.print(w, opts, widths)
		}
	}
}

// print outputs an entry with the specified formatting options.
func (e *entry) print(w io.Writer, opts options, widths widths) {
	var column []string

	// Add a uniform-width column for each requested property.
	if opts.mode {
		column = append(column, fmt.Sprintf("%*s", widths.mode, e.Mode))
	}
	if opts.user {
		column = append(column, fmt.Sprintf("%*s", widths.user, e.User))
	}
	if opts.group {
		column = append(column, fmt.Sprintf("%*s", widths.group, e.Group))
	}
	if opts.size {
		column = append(column, fmt.Sprintf("%*d", widths.size, e.Size))
	}
	if opts.inode {
		column = append(column, fmt.Sprintf("%*d", widths.inode, e.Inode))
	}
	if opts.mount {
		var ind string
		if e.Dev != e.Pdev {
			ind = mountPointSymbol
		}
		column = append(column, fmt.Sprintf("%*s", len(mountPointSymbol), ind))
	}

	// Append the indented name (with possible link target).
	name := e.Name
	if !opts.noFollow {
		name = e.fmtName()
	}

	// Join columns together with separator.
	fmt.Fprintln(w, strings.Join(append(column, name), " "))
}
