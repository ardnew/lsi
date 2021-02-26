package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ardnew/lsi"
	"github.com/ardnew/version"
	"github.com/juju/errors"
)

func init() {
	version.ChangeLog = []version.Change{{
		Package: "lsi",
		Version: "0.1.0",
		Date:    "2021 Feb 22",
		Description: []string{
			"initial implementation",
		},
	}}
}

func usage() {
	exe, err := os.Executable()
	if nil != err {
		panic(errors.Annotate(err, "os.Executable()"))
	}
	exe, err = filepath.EvalSymlinks(exe)
	if nil != err {
		panic(errors.Annotatef(err, "filepath.EvalSymlinks(%q)", exe))
	}
	exe = filepath.Base(exe)
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  "+exe, "[flags]", "[--]", "[PATH ...]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "flags:")
	fmt.Fprintln(os.Stderr, "  -v  Display version information.")
	fmt.Fprintln(os.Stderr, "  -a  Display change history.")
	fmt.Fprintln(os.Stderr, "  -n  Do not follow symlinks.")
	fmt.Fprintln(os.Stderr, "  -l  Output using long format (-p -u -g -s -m).")
	fmt.Fprintln(os.Stderr, "  -p  Output file type and permissions.")
	fmt.Fprintln(os.Stderr, "  -u  Output file owner.")
	fmt.Fprintln(os.Stderr, "  -g  Output file group.")
	fmt.Fprintln(os.Stderr, "  -s  Output file size (bytes).")
	fmt.Fprintln(os.Stderr, "  -i  Output file inode.")
	fmt.Fprintln(os.Stderr, "  -m  Output mount point symbols ("+mountPointSymbol+").")
}

const mountPointSymbol = "@"

func main() {

	var (
		changesFlag  bool
		versionFlag  bool
		noFollowFlag bool
		longFlag     bool
		modeFlag     bool
		userFlag     bool
		groupFlag    bool
		sizeFlag     bool
		inodeFlag    bool
		mountFlag    bool
	)

	flag.BoolVar(&changesFlag, "a", false, "")
	flag.BoolVar(&versionFlag, "v", false, "")
	flag.BoolVar(&noFollowFlag, "n", false, "")
	flag.BoolVar(&longFlag, "l", false, "")
	flag.BoolVar(&modeFlag, "p", false, "")
	flag.BoolVar(&userFlag, "u", false, "")
	flag.BoolVar(&groupFlag, "g", false, "")
	flag.BoolVar(&sizeFlag, "s", false, "")
	flag.BoolVar(&inodeFlag, "i", false, "")
	flag.BoolVar(&mountFlag, "m", false, "")
	flag.Usage = usage
	flag.Parse()

	// If version or changelog requested, print the output and exit immediately.
	if changesFlag {
		version.PrintChangeLog()
	} else if versionFlag {
		fmt.Printf("lsi version %s\n", version.String())
	} else {
		// Determine the file paths to analyze.
		path := []string{}
		if 0 == flag.NArg() {
			// If no paths were given, use PWD.
			wd, err := os.Getwd()
			if nil != err {
				panic(errors.Annotate(err, "os.Getwd()"))
			}
			path = append(path, wd)
		} else {
			path = append(path, flag.Args()...)
		}

		// Configure the meta-flags.
		if longFlag {
			modeFlag, userFlag, groupFlag, sizeFlag, mountFlag =
				true, true, true, true, true
		}

		// Repeat the following for each path provided.
		numPaths := len(path)
		for i, p := range path {

			// If more than one path provided, print a header for the current path.
			if numPaths > 1 {
				fp := filepath.Clean(p)
				fmt.Println("--", fp)
			}

			comp := []lsi.Component{}
			var modeLen, userLen, groupLen, sizeLen, inodeLen colsz

			// Perform the walk, building a buffer of all discovered path elements.
			_ = lsi.Walk(p, func(c lsi.Component) (bool, error) {
				comp = append(comp, c)
				// Stop building buffer on the first error encountered.
				if c.Err != nil {
					return false, c.Err
				}
				// Keep track of the minimum width for each column.
				modeLen.set(len(c.Mode))
				userLen.set(len(c.User))
				groupLen.set(len(c.Group))
				sizeLen.set(len(strconv.FormatInt(c.Size, 10)))
				inodeLen.set(len(strconv.FormatUint(c.Inode, 10)))
				return !noFollowFlag, nil
			})

			// Print the buffer of discovered path elements.
			for _, c := range comp {
				var str string
				if c.Err != nil {
					// If one of the path elements contained an error, try to provide
					// context based on the type of error generated.
					switch err := c.Err.(type) {
					case *os.PathError:
						str = fmt.Sprintf(" * %s (%s): %s", c.Name, err.Path, err.Err)
					default:
						str = fmt.Sprintf(" * %s: %s", c.Name, err)
					}
				} else {
					column := []string{}
					// Add a uniform-width column for each requested property.
					if modeFlag {
						column = append(column, fmt.Sprintf("%*s", modeLen, c.Mode))
					}
					if userFlag {
						column = append(column, fmt.Sprintf("%*s", userLen, c.User))
					}
					if groupFlag {
						column = append(column, fmt.Sprintf("%*s", groupLen, c.Group))
					}
					if sizeFlag {
						column = append(column, fmt.Sprintf("%*d", sizeLen, c.Size))
					}
					if inodeFlag {
						column = append(column, fmt.Sprintf("%*d", inodeLen, c.Inode))
					}
					if mountFlag {
						var ind string
						if c.Dev != c.Pdev {
							ind = mountPointSymbol
						}
						column = append(column, fmt.Sprintf("%*s", len(mountPointSymbol), ind))
					}
					// Append the indented name (with possible link target).
					name := c.Name
					if !noFollowFlag {
						name = c.FmtName()
					}
					// Join columns together with separator.
					str = strings.Join(append(column, name), " ")
				}
				fmt.Println(str)
			}

			if numPaths > 1 && i+1 < numPaths {
				fmt.Println()
			}
		}
	}
}

type colsz int

func (c *colsz) set(n int) {
	if n > int(*c) {
		*c = colsz(n)
	}
}
