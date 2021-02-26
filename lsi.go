package lsi

import (
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

var IndentWidth = 2

type WalkFunc func(c Component) (follow bool, err error)

func Walk(path string, fn WalkFunc) error {
	return walk(path, 0, fn)
}

type Component struct {
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

func MakeComponent(path, volume, name string, level int) Component {
	var (
		link, mod, usr, grp string
		uid, gid            int
		dev, pdev, inode    uint64
		size                int64
	)

	info, err := os.Lstat(path)
	if nil == err {
		// If given path is a symlink, try to determine its target.
		if 0 != info.Mode()&fs.ModeSymlink {
			link, err = os.Readlink(path)
		}
		// Create a symbolic mode string.
		mod = mode(info)
	}

	pdev = ^uint64(0) // Surely no device would really have this...
	if nil == err {
		// Try to determine our parent's device ID. If this is different from our
		// own, then we can assume we are a mount point.
		if abs, aerr := filepath.Abs(path); nil == aerr {
			if parent := filepath.Dir(abs); parent != path {
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
		// Now try to determine the given path's owner and group.
		// The following will be OS-specific and has not been tested anywhere
		// other than Linux.
		switch stat := info.Sys().(type) {
		case *syscall.Stat_t:
			dev, inode, size = stat.Dev, stat.Ino, stat.Size
			uid, gid = int(stat.Uid), int(stat.Gid)
			// We must convert the integer UID/GID to decimal strings for the OS or
			// syscall API. No idea why, but them's the rules.
			ustr := strconv.FormatInt(int64(uid), 10)
			if u, e := user.LookupId(ustr); nil != e {
				err = e
				break
			} else {
				usr = u.Username // u.User contains the user's fullname, not username
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

	return Component{
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

func (c *Component) String() string {
	if c.Err == nil {
		return fmt.Sprintf("%s %s %s %s", c.Mode, c.User, c.Group, c.FmtName())
	}
	return fmt.Sprintf(" * %s: %s", c.Name, c.Err)
}

func (c *Component) FmtName() string {
	var link string
	if c.Link != "" {
		link = " -> " + c.Link
	}
	return fmt.Sprintf("%*s%s%s", IndentWidth*c.Level, "", c.Name, link)
}

// mode returns a symbolic string representation for the filesystem attributes
// of the file described by the given os.FileInfo.
//
// Note that the symbols used here are GNU coreutils convention (the same ones
// used by `ls`) and not the ones used by standard Go (package "io/fs").
//
// Because of this, we use only a single rune for the type attributes (the
// leading rune, specifically), and we combine the setuid/setgid/sticky symbols
// with the permission symbols the same way `ls` does.
//
// Furthermore, the following extended attributes all share a common symbol '+'
// that is appended to the end of the mode string: Append-only, Exclusive Use,
// Temporary File. Since these attributes are not commonly-encountered, and they
// do not affect behavior until the user attempts I/O on its content, we simply
// provide the symbol as a hint that other attributes exist and may need to be
// considered (e.g., `chattr` and family).
//
// These deviations from standard Go mean we always return a mode string that is
// exactly 10 (or 11 if xattr symbol '+' appended) bytes long. In contrast, for
// example, Go produces mode strings of variable length depending on the number
// of attribute bits that are set. But since the variable-length portion of the
// string is *prepended* to the front of the string, it makes for ugly/complex
// formatting when trying to align columns in any output.
func mode(info os.FileInfo) string {

	// Start with the full-width mode string, and modify runes at specific index
	// to build the result.
	s := []rune("-rwxrwxrwx")
	m := info.Mode()

	// For any bit not set in the lower-9 bits (permissions), replace the
	// corresponding rune with a dash.
	for i := len(s) - 1; i > 0; i-- {
		if 0 == m&(1<<uint(i-1)) {
			s[len(s)-i] = '-'
		}
	}

	// For each of the file attribute bits that describe the type of file, replace
	// the leading rune of the mode string with a representative symbol.

	const attr = "d---lbps--c-?" // standard Go = "dalTLDpSugct?"
	for i, c := range attr {
		if (c != '-') && (m&(1<<uint(32-1-i)) != 0) {
			s[0] = c
		}
	}

	if 0 != m&fs.ModeSetuid {
		s[3] = mapRune('s', 0 == m&(1<<8)) // if mode & 0400 then 's', else 'S'
	}
	if 0 != m&fs.ModeSetgid {
		s[6] = mapRune('s', 0 == m&(1<<5)) // if mode & 0040 then 's', else 'S'
	}
	if 0 != m&fs.ModeSticky {
		s[9] = mapRune('t', 0 == m&(1<<2)) // if mode & 0004 then 't', else 'T'
	}

	if 0 != m&(fs.ModeAppend|fs.ModeExclusive|fs.ModeTemporary) {
		s = append(s, '+') // catch-all indicator for extended attributes
	}

	return string(s)
}

func walk(path string, level int, fn WalkFunc) error {

	var elem []string
	var volume string

	// Always reduce the path lexically. This does not use the actual filesystem
	// in any way, but it does remove garbage that isn't useful to anyone trying
	// to analyze a given path.
	path = filepath.Clean(path)

	// The difference in handling between relative and absolute file paths is
	// subtle, but it's complicated by the ways they can be expressed on different
	// systems (Unix vs. Windows).
	// So we first separate the parsing based on path relativity, and then we
	// handle the path based on target system.
	if filepath.IsAbs(path) {
		var root string // drive/UNC volume on Windows, otherwise "/"
		var start int   // index of first byte following root separator

		// Check if we have a volume, e.g., drive letter or UNC share, which is only
		// present on Windows. Otherwise, VolumeName always returns an empty string.
		if vol := filepath.VolumeName(path); vol != "" {
			volume = vol
			root = vol + string(filepath.Separator)
			start = len(vol) + 1
		} else {
			root = string(filepath.Separator)
			start = 1
		}

		// Set the volume/root as first element, and append any remaining trailing
		// path components.
		elem = []string{root}
		if len(path) > start {
			elem = append(elem, strings.Split(path[start:], string(filepath.Separator))...)
		}
	} else {
		var start int // index of first byte following volume separator

		// Check if we have a volume, e.g., drive letter or UNC share, which is only
		// present on Windows. Otherwise, VolumeName always returns an empty string.
		if vol := filepath.VolumeName(path); vol != "" {
			volume = vol
			start = len(vol)
		}

		// For relative paths (which have been lexically cleaned), we can simply
		// split the path into components delimited by the directory separator.
		if len(path) > start {
			elem = strings.Split(path, string(filepath.Separator))
		}
	}

	// Create a Component and invoke user callback for each path from elem.
	for i, e := range elem {
		c := MakeComponent(filepath.Join(elem[:i+1]...), volume, e, level)
		// Invoke user callback to determine if processing should continue.
		// If continuing with a symlink, the callback also determines if we should
		// follow the symlink to its target.
		follow, err := fn(c)
		// Stop processing and return immediately if callback returns an error.
		if nil != err {
			return err
		}
		// If component is a symlink and callback allows it, traverse its target.
		if follow && c.Link != "" {
			if err := walk(c.Link, level+1, fn); nil != err {
				return err
			}
		}
	}

	return nil
}

// mapRune returns the given rune as uppercase if and only if upper is true,
// otherwise it returns the given rune as lowercase.
func mapRune(c rune, upper bool) rune {
	if upper {
		return unicode.ToUpper(c)
	}
	return unicode.ToLower(c)
}
