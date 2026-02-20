//go:build windows

package main

import (
	"os"
)

// getDeviceInfo extracts device, inode, and size from file info (Windows stub).
func getDeviceInfo(info os.FileInfo) (dev, inode uint64, size int64) {
	// Windows doesn't have the same device/inode concept
	size = info.Size()
	return
}

// getParentDevice gets the device ID of the parent directory (Windows stub).
func getParentDevice(dest string) uint64 {
	return ^uint64(0) // Invalid device ID
}

// getOwnerInfo gets user and group information (Windows stub).
func getOwnerInfo(info os.FileInfo) (usr, grp string, uid, gid int, err error) {
	// Windows user/group information requires different API
	// Return empty values for now
	return
}
