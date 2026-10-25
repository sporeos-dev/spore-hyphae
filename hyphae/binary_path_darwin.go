// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build darwin

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

func binaryPathForPID(pid int) (string, error) {
	buf := make([]byte, 4096)
	ret, _, errno := syscall.Syscall(syscall.SYS_PROC_INFO,
		uintptr(9),
		uintptr(pid),
		uintptr(unsafe.Pointer(&buf[0])),
	)
	if errno != 0 || ret == 0 {
		return "", fmt.Errorf("proc_pidpath failed for pid %d: %w", pid, errno)
	}
	n := ret
	for i := uintptr(0); i < ret; i++ {
		if buf[i] == 0 {
			n = i
			break
		}
	}
	return string(buf[:n]), nil
}
