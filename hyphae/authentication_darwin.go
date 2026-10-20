// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build darwin && cgo

package main

/*
#cgo LDFLAGS: -framework Foundation -framework LocalAuthentication
#include <stdlib.h>

int spore_authenticate(const char *reason, char **error_message);
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func authenticateUser(reason string) (bool, error) {
	cReason := C.CString(reason)
	defer C.free(unsafe.Pointer(cReason))

	var cError *C.char
	outcome := C.spore_authenticate(cReason, &cError)
	if cError != nil {
		defer C.free(unsafe.Pointer(cError))
	}

	switch outcome {
	case 1:
		return true, nil
	case 0:
		return false, nil
	default:
		return false, fmt.Errorf("macOS authentication unavailable: %s", C.GoString(cError))
	}
}
