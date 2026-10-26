// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func authenticateUser(reason string) (bool, error) {
	script := `Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; $f=New-Object Windows.Forms.Form; $f.Text='Spore confirmation'; $f.Size=New-Object Drawing.Size(460,220); $f.StartPosition='CenterScreen'; $l=New-Object Windows.Forms.Label; $l.Text=$env:SPORE_CONFIRMATION_REASON + [Environment]::NewLine + [Environment]::NewLine + 'Type yes to continue.'; $l.AutoSize=$true; $l.MaximumSize=New-Object Drawing.Size(420,0); $l.Location=New-Object Drawing.Point(15,15); $t=New-Object Windows.Forms.TextBox; $t.Location=New-Object Drawing.Point(15,120); $t.Width=410; $b=New-Object Windows.Forms.Button; $b.Text='Continue'; $b.Location=New-Object Drawing.Point(350,150); $b.Enabled=$false; $t.Add_TextChanged({$b.Enabled=($t.Text -ceq 'yes')}); $b.Add_Click({$f.DialogResult='OK';$f.Close()}); $f.Controls.AddRange(@($l,$t,$b)); if($f.ShowDialog() -eq 'OK'){[Console]::Out.Write('yes')}`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-STA", "-Command", script)
	cmd.Env = append(os.Environ(), "SPORE_CONFIRMATION_REASON="+reason)
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("Windows confirmation dialog failed: %w", err)
	}
	return string(out) == "yes", nil
}
