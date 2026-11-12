package app

import (
	"syscall"
	"unsafe"
)

func diskFreeMB(path string) int64 {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")

	var freeBytesAvailable uint64
	pathPtr, _ := syscall.UTF16PtrFromString(path)
	ret, _, _ := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		0, 0,
	)
	if ret == 0 {
		return -1
	}
	return int64(freeBytesAvailable / 1024 / 1024)
}
