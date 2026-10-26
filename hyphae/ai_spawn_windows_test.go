// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build windows

package main

import (
	"os/exec"
	"testing"
)

func TestConfigureSpawnCommandHidesWindow(t *testing.T) {
	cmd := exec.Command("cmd.exe")

	configureSpawnCommand(cmd)

	if cmd.SysProcAttr == nil {
		t.Fatal("configureSpawnCommand() did not configure process attributes")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Error("configureSpawnCommand() did not hide the spawned process window")
	}
}
