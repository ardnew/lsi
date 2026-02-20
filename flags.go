package main

import (
	"fmt"
	"io"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/integrii/flaggy"
)

const (
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

// parseFlags parses command-line arguments and returns options and remaining paths.
// It returns an error if flag parsing fails.
func parseFlags(args []string) (opts options, paths []string, err error) {
	// Use panic instead of os.Exit() for testability.
	// We recover from panics and return errors instead.
	// This must be set before creating the parser.
	origPanicSetting := flaggy.PanicInsteadOfExit
	flaggy.PanicInsteadOfExit = true
	defer func() {
		flaggy.PanicInsteadOfExit = origPanicSetting
	}()

	// Create a new parser for testability (avoid global state).
	parser := flaggy.NewParser(command)
	parser.Description = "Analyze file paths by traversing and displaying each path component"
	parser.Version = getVersion()

	// Disable flaggy's built-in version handling to preserve testability.
	// We handle version ourselves in run().
	parser.ShowVersionWithVersionFlag = false

	// Disable automatic help display to prevent os.Exit() calls during testing.
	parser.ShowHelpWithHFlag = false

	// Allow unregistered trailing arguments (paths) without requiring --.
	// This means unknown flags will be treated as paths rather than errors,
	// but provides better UX for the primary use case.
	parser.ShowHelpOnUnexpected = false

	// Register flags with short and long names.
	parser.Bool(&opts.version, "v", "version", "Display version information")
	parser.Duration(&opts.timeout, "t", "timeout", "Timeout duration (e.g., 30s, 5m)")
	parser.Bool(&opts.noFollow, "n", "no-follow", "Do not follow symlinks")
	parser.Bool(&opts.long, "l", "long", "Output using long format (-p -u -g -s -m)")
	parser.Bool(&opts.mode, "p", "permissions", "Output file type and permissions")
	parser.Bool(&opts.user, "u", "user", "Output file owner")
	parser.Bool(&opts.group, "g", "group", "Output file group")
	parser.Bool(&opts.size, "s", "size", "Output file size (bytes)")
	parser.Bool(&opts.inode, "i", "inode", "Output file inode")
	parser.Bool(&opts.mount, "m", "mount", "Output mount point symbols ("+mountPointSymbol+")")

	// Recover from panics that flaggy might trigger for invalid input.
	defer func() {
		if r := recover(); r != nil {
			// Convert panic to error for proper error handling.
			err = fmt.Errorf("flag parsing error: %v", r)
			opts = options{}
			paths = nil
		}
	}()

	// Parse the arguments.
	if parseErr := parser.ParseArgs(args); parseErr != nil {
		return options{}, nil, parseErr
	}

	// Configure the meta-flags.
	if opts.long {
		opts.mode, opts.user, opts.group, opts.size, opts.mount = true, true, true, true, true
	}

	// Return options and trailing arguments (paths).
	return opts, parser.TrailingArguments, nil
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

// showHelp displays the help message for lsi.
func showHelp(w io.Writer) {
	fmt.Fprintf(w, "%s - Analyze file paths by traversing and displaying each path component\n\n", command)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintf(w, "  %s [flags] [--] [PATH ...]\n\n", command)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  -h --help          Display this help message")
	fmt.Fprintln(w, "  -v --version       Display version information")
	fmt.Fprintln(w, "  -t --timeout       Timeout duration (e.g., 30s, 5m)")
	fmt.Fprintln(w, "  -n --no-follow     Do not follow symlinks")
	fmt.Fprintln(w, "  -l --long          Output using long format (-p -u -g -s -m)")
	fmt.Fprintln(w, "  -p --permissions   Output file type and permissions")
	fmt.Fprintln(w, "  -u --user          Output file owner")
	fmt.Fprintln(w, "  -g --group         Output file group")
	fmt.Fprintln(w, "  -s --size          Output file size (bytes)")
	fmt.Fprintln(w, "  -i --inode         Output file inode")
	fmt.Fprintf(w, "  -m --mount         Output mount point symbols (%s)\n", mountPointSymbol)
}
