// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !darwin || !cgo

package main

func authenticateUser(reason string) (bool, error) {
	return unsupportedAuthentication(reason)
}
