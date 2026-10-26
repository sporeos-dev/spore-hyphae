// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build windows

package main

import (
	"os"
	"testing"
)

func TestWindowsAuthenticationSmoke(t *testing.T) {
	if os.Getenv("SPORE_WINDOWS_AUTH_SMOKE") != "1" {
		t.Skip("set SPORE_WINDOWS_AUTH_SMOKE=1 to verify the Windows credential prompt")
	}

	confirmed, err := authenticateUser("Spore Hyphae test: enter your Windows password or cancel this request.")
	if err != nil {
		t.Fatalf("Windows authentication failed: %v", err)
	}
	t.Logf("Windows authentication prompt completed; confirmed=%t", confirmed)
}
