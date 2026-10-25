// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux

package main

import (
	"testing"
)

func TestPolkitDecision(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		wantYes  bool
		wantErr  bool
	}{
		{name: "authorized", exitCode: 0, wantYes: true},
		{name: "denied", exitCode: 1},
		{name: "dismissed", exitCode: 3},
		{name: "no agent", exitCode: 2, wantErr: true},
		{name: "command error", exitCode: 127, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotYes, err := polkitDecision(test.exitCode)
			if gotYes != test.wantYes || (err != nil) != test.wantErr {
				t.Fatalf("polkitDecision(%d) = (%t, %v), want (%t, error=%t)",
					test.exitCode, gotYes, err, test.wantYes, test.wantErr)
			}
		})
	}
}

func TestLinuxProcessStatFieldsHandlesCommandPunctuation(t *testing.T) {
	stat := "42 (hyphae ) with spaces) S 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 987654 21"
	fields, ok := linuxProcessStatFields(stat)
	if !ok {
		t.Fatal("linuxProcessStatFields() rejected a valid stat record")
	}
	if len(fields) <= 19 || fields[19] != "987654" {
		t.Fatalf("start-time field = %v, want 987654 at index 19", fields)
	}
}
