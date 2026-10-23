// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !linux && (!darwin || !cgo)

package main

import (
	"fmt"
	"runtime"
)

func binaryPathForPID(pid int) (string, error) {
	return "", fmt.Errorf("binaryPathForPID: unsupported platform %s", runtime.GOOS)
}