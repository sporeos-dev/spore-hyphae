// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const polkitActionID = "dev.sporeos.hyphae.confirm"

func authenticateUser(_ string) (bool, error) {
	// Polkit rejects caller-supplied details from untrusted user processes.
	// The installed action's fixed message is used instead of forwarding the
	// node/command text as details, which caused pkcheck to exit with 127.
	checker, err := findPolkitChecker()
	if err != nil {
		return false, err
	}

	startTime, err := linuxProcessStartTime(os.Getpid())
	if err != nil {
		return false, fmt.Errorf("could not identify the Hyphae process for Polkit: %w", err)
	}
	subject := fmt.Sprintf("%d,%s,%d", os.Getpid(), startTime, os.Getuid())
	args := []string{
		"--action-id", polkitActionID,
		"--process", subject,
		"--allow-user-interaction",
	}

	output, err := exec.Command(checker, args...).CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return false, fmt.Errorf("could not contact Polkit: %w: %s", err, strings.TrimSpace(string(output)))
		}
		confirmed, decisionErr := polkitDecision(exitErr.ExitCode())
		if decisionErr != nil && len(strings.TrimSpace(string(output))) > 0 {
			return false, fmt.Errorf("%w: %s", decisionErr, strings.TrimSpace(string(output)))
		}
		return confirmed, decisionErr
	}
	return true, nil
}

func findPolkitChecker() (string, error) {
	for _, path := range []string{"/usr/bin/pkcheck", "/bin/pkcheck"} {
		info, err := os.Stat(path)
		if err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0 {
			return path, nil
		}
	}
	return "", errors.New("Polkit checker (pkcheck) is unavailable")
}

func linuxProcessStartTime(pid int) (string, error) {
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return "", err
	}
	fields, ok := linuxProcessStatFields(string(stat))
	if !ok || len(fields) <= 19 {
		return "", errors.New("unexpected /proc process stat format")
	}
	if _, err := strconv.ParseUint(fields[19], 10, 64); err != nil {
		return "", fmt.Errorf("invalid process start time: %w", err)
	}
	return fields[19], nil
}

func linuxProcessStatFields(stat string) ([]string, bool) {
	// The comm field is parenthesized and may itself contain spaces or ')'.
	closeParen := strings.LastIndex(stat, ")")
	if closeParen < 0 || closeParen+1 >= len(stat) {
		return nil, false
	}
	return strings.Fields(stat[closeParen+1:]), true
}

func polkitDecision(exitCode int) (bool, error) {
	switch exitCode {
	case 0:
		return true, nil
	case 1, 3:
		// Not authorized or the user dismissed the authentication dialog.
		return false, nil
	case 2:
		return false, errors.New("Polkit could not authorize the request: no authentication agent is available")
	default:
		return false, fmt.Errorf("Polkit authorization check failed with exit code %d", exitCode)
	}
}
