// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build windows

package main

import winapi "golang.org/x/sys/windows"

func binaryPathForPID(pid int) (string, error) {
	process, err := winapi.OpenProcess(winapi.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return "", err
	}
	defer winapi.CloseHandle(process)

	path := make([]uint16, winapi.MAX_LONG_PATH)
	pathLength := uint32(len(path))
	if err := winapi.QueryFullProcessImageName(process, 0, &path[0], &pathLength); err != nil {
		return "", err
	}
	return winapi.UTF16ToString(path[:pathLength]), nil
}
