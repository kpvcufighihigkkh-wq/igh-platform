//go:build !windows

package app

import "syscall"

func diskFreeMB(path string) int64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return -1
	}
	return int64(stat.Bavail*uint64(stat.Bsize)) / 1024 / 1024
}
