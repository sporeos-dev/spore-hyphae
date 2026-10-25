// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build windows

package main

import "testing"

func TestWindowsConsentResultMapping(t *testing.T) {
	verified, err := interpretConsentResult(consentVerified)
	if err != nil || !verified {
		t.Fatalf("verified result = (%t, %v), want (true, nil)", verified, err)
	}

	verified, err = interpretConsentResult(consentCanceled)
	if err != nil || verified {
		t.Fatalf("canceled result = (%t, %v), want (false, nil)", verified, err)
	}

	if _, err := interpretConsentResult(consentNotConfiguredForUser); err == nil {
		t.Fatal("unconfigured result unexpectedly succeeded")
	}
}
