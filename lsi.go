package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unicode"
)

const indentWidth = 2

// walkFunc is called for each path element discovered during traversal.
type walkFunc func(ctx context.Context, e entry) (follow bool, err error)

// walk traverses the given path, invoking fn for each element encountered.
func walk(ctx context.Context, path string, fn walkFunc) error {
	return walkRecursive(ctx, "", path, 0, fn)
}

// entry represents a single path element with its associated metadata.
type entry struct {
	Path   string
	Volume string
	Name   string
	Link   string
	Mode   string
	Dev    uint64
	Pdev   uint64
	Inode  uint64
	Size   int64
	Uid    int
	User   string
	Gid    int
	Group  string
	Level  int
	Info   os.FileInfo
	Err    error
}

// makeEntry creates an entry for the given path component.
func makeEntry(ctx context.Context, from, path, volume, name string, level int) entry {
	// Check for context cancellation early.
	if ctx.Err() != nil {
		return entry{
			Path:   path,
			Volume: volume,
			Name:   name,
			Level:  level,
			Err:    ctx.Err(),
		}
	}

	var (
		link, mod, usr, grp string
		uid, gid            int
		dev, pdev, inode    uint64
		size                int64
	)

	// Build paths relative to where we are from.
	dest := filepath.Join(from, path)

	info, err := os.Lstat(dest)
	if nil == err {
		// If given path is a symlink, determine its target.
		if 0 != info.Mode()&fs.ModeSymlink {
			link, err = os.Readlink(dest)
		}
		mod = mode(info)
	}

	pdev = ^uint64(0) // Surely no device would really have this...
	if nil == err {
		// Determine parent's device ID to detect mount points.
		if abs, aerr := filepath.Abs(dest); nil == aerr {
			if parent := filepath.Dir(abs); parent != dest {
				if pinfo, perr := os.Stat(parent); nil == perr {
					switch stat := pinfo.Sys().(type) {
					case *syscall.Stat_t:
						pdev = stat.Dev
					}
				}
			}
		}
	}

	if nil == err {
		// Determine owner and group (OS-specific, tested on Linux).
		switch stat := info.Sys().(type) {
		case *syscall.Stat_t:
			dev, inode, size = stat.Dev, stat.Ino, stat.Size
			uid, gid = int(stat.Uid), int(stat.Gid)
			ustr := strconv.FormatInt(int64(uid), 10)
			if u, e := user.LookupId(ustr); nil != e {
				err = e
				break
			} else {
				usr = u.Username
			}
			gstr := strconv.FormatInt(int64(gid), 10)
			if g, e := user.LookupGroupId(gstr); nil != e {
				err = e
				break
			} else {
				grp = g.Name
			}
		}
	}

	return entry{
		Path:   path,
		Volume: volume,
		Name:   name,
		Link:   link,
		Mode:   mod,
		Dev:    dev,
		Pdev:   pdev,
		Inode:  inode,
		Size:   size,
		Uid:    uid,
		User:   usr,
		Gid:    gid,
		Group:  grp,
		Level:  level,
		Info:   info,
		Err:    err,
	}
}

// fmtName returns the entry name with indentation and link target if applicable.
func (e *entry) fmtName() string {
	var link string
	if e.Link != "" {
		link = " -> " + e.Link
	}
	return fmt.Sprintf("%*s%s%s", indentWidth*e.Level, "", e.Name, link)
}

// mode returns a symbolic string representation for filesystem attributes.
// Uses GNU coreutils convention (like `ls`) rather than standard Go.
func mode(info os.FileInfo) string {
	s := []rune("-rwxrwxrwx")
	m := info.Mode()

	// Replace unset permission bits with dashes.
	for i := len(s) - 1; i > 0; i-- {
		if 0 == m&(1<<uint(i-1)) {
			s[len(s)-i] = '-'
		}
	}

	// Set the leading rune based on file type.
	const attr = "d---lbps--c-?" // standard Go = "dalTLDpSugct?"
	for i, c := range attr {
		if (c != '-') && (m&(1<<uint(32-1-i)) != 0) {
			s[0] = c
		}
	}

	if 0 != m&fs.ModeSetuid {
		s[3] = upperIf('s', 0 == m&(1<<8))
	}
	if 0 != m&fs.ModeSetgid {
		s[6] = upperIf('s', 0 == m&(1<<5))
	}
	if 0 != m&fs.ModeSticky {
		s[9] = upperIf('t', 0 == m&(1<<2))
	}

	if 0 != m&(fs.ModeAppend|fs.ModeExclusive|fs.ModeTemporary) {
		s = append(s, '+')
	}

	return string(s)
}

// splitPath separates a path into its volume and element components.
func splitPath(path string) (elem []string, volume string) {
	// Always reduce the path lexically.
	path = filepath.Clean(path)

	// Handle absolute and relative paths differently.
	if filepath.IsAbs(path) {
		var root string
		var start int

		if vol := filepath.VolumeName(path); vol != "" {
			volume = vol
			root = vol + string(filepath.Separator)
			start = len(vol) + 1
		} else {
			root = string(filepath.Separator)
			start = 1
		}

		elem = []string{root}
		if len(path) > start {
			elem = append(elem, strings.Split(path[start:], string(filepath.Separator))...)
		}
	} else {
		var start int

		if vol := filepath.VolumeName(path); vol != "" {
			volume = vol
			start = len(vol)
		}

		if len(path) > start {
			elem = strings.Split(path, string(filepath.Separator))
		}
	}

	return
}

// walkRecursive recursively traverses path elements.
func walkRecursive(ctx context.Context, from, path string, level int, fn walkFunc) error {
	// Check for context cancellation.
	if ctx.Err() != nil {
		return ctx.Err()
	}

	elem, volume := splitPath(path)

	// Invoke callback for each path element.
	for i, name := range elem {
		e := makeEntry(ctx, from, filepath.Join(elem[:i+1]...), volume, name, level)

		follow, err := fn(ctx, e)
		if nil != err {
			return err
		}

		// If entry is a symlink and callback allows it, traverse its target.
		if follow && e.Link != "" {
			var rel string
			if !filepath.IsAbs(e.Link) {
				rel = filepath.Join(from, filepath.Join(elem[:i]...))
			}
			if err := walkRecursive(ctx, rel, e.Link, level+1, fn); nil != err {
				return err
			}
		}
	}

	return nil
}

// upperIf returns the rune as uppercase if upper is true, otherwise lowercase.
func upperIf(c rune, upper bool) rune {
	if upper {
		return unicode.ToUpper(c)
	}
	return unicode.ToLower(c)
}
