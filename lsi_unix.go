//go:build unix

package main

import (
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
)

// getDeviceInfo extracts device, inode, and size from file info (Unix-specific).
func getDeviceInfo(info os.FileInfo) (dev, inode uint64, size int64) {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		dev = uint64(stat.Dev)
		inode = stat.Ino
		size = stat.Size
	}
	return
}

// getParentDevice gets the device ID of the parent directory (Unix-specific).
func getParentDevice(dest string) uint64 {
	pdev := ^uint64(0) // Default: invalid device ID
	if abs, err := filepath.Abs(dest); err == nil {
		if parent := filepath.Dir(abs); parent != dest {
			if pinfo, err := os.Stat(parent); err == nil {
				if stat, ok := pinfo.Sys().(*syscall.Stat_t); ok {
					pdev = uint64(stat.Dev)
				}
			}
		}
	}
	return pdev
}

// getOwnerInfo gets user and group information (Unix-specific).
func getOwnerInfo(info os.FileInfo) (usr, grp string, uid, gid int, err error) {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		uid = int(stat.Uid)
		gid = int(stat.Gid)

		ustr := strconv.FormatInt(int64(uid), 10)
		if u, e := user.LookupId(ustr); e != nil {
			err = e
			return
		} else {
			usr = u.Username
		}

		gstr := strconv.FormatInt(int64(gid), 10)
		if g, e := user.LookupGroupId(gstr); e != nil {
			err = e
			return
		} else {
			grp = g.Name
		}
	}
	return
}
