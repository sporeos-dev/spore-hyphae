// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package main

import "fmt"

func unsupportedAuthentication(reason string) (bool, error) {
	return false, fmt.Errorf("OS-authenticated confirmation is unsupported on this platform")
}
