// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build darwin && cgo

package main

/*
#cgo LDFLAGS: -lproc
#include <libproc.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func binaryPathForPID(pid int) (string, error) {
	buffer := make([]byte, C.PROC_PIDPATHINFO_MAXSIZE)
	length := C.proc_pidpath(C.int(pid), unsafe.Pointer(&buffer[0]), C.uint32_t(len(buffer)))
	if length <= 0 {
		return "", fmt.Errorf("proc_pidpath failed for pid %d", pid)
	}
	return C.GoString((*C.char)(unsafe.Pointer(&buffer[0]))), nil
}
