// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build windows

package main

import (
	"os"
	"testing"
)

func TestWindowsBinaryPathForCurrentProcess(t *testing.T) {
	path, err := binaryPathForPID(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	actual, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	expectedPath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.Stat(expectedPath)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(actual, expected) {
		t.Fatalf("binaryPathForPID() = %q, want %q", path, expectedPath)
	}
}
