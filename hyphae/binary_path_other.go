// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !darwin && !linux && !windows

package main

import "fmt"

func binaryPathForPID(pid int) (string, error) {
	return "", fmt.Errorf("binaryPathForPID: unsupported platform")
}
