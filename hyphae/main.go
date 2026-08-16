// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"syscall"
	"unsafe"

	"github.com/kardianos/service"
	spore "github.com/sporeos-dev/spore-client-libs/spore_go"
	"github.com/sporeos-dev/spore-client-libs/spore_go/request"
	"github.com/sporeos-dev/spore-client-libs/spore_go/response"
)

const (
	errArgumentMissing     = "RequiredArgumentMissing"
	errArgumentInvalidType = "ArgumentInvalidType"
	errRuntime             = "Runtime"
)

const appId = "dev.sporeos.hyphae"

var svcConfig = &service.Config{
	Name:        "dev.sporeos.agent",
	DisplayName: "Spore Agent",
	Description: "Extends the reach of the hub into the user space.",
	// UserService installs to ~/Library/LaunchAgents (macOS) and
	// ~/.config/systemd/user (Linux) — no elevated privileges required.
	Option: service.KeyValue{
		"UserService": true,
	},
}

// program implements service.Interface. It owns the spore client and
// manages connection lifetime.
type program struct {
	client *spore.Client
}

// Start is called by the service manager on launch.
func (p *program) Start(s service.Service) error {
	client := spore.New(appId).
		WithDefaultErrorHandler()
	p.client = client

	// Dispatch incoming hub requests by command name.
	p.client.OnRequest(func(req *request.Request) {
		switch req.Command() {
		case "HYPHAE.manifest.read":
			p.handleManifestRead(req)
		case "HYPHAE.binary.hash":
			p.handleBinaryHash(req)
		case "HYPHAE.file.hash":
			p.handleFileHash(req)
		case "HYPHAE.node.spawn":
			p.handleSpawn(req)
		case "HYPHAE.node.kill":
			p.handleKill(req)
		}
	})

	go p.run()
	return nil
}

// run connects to the hub and listens, reconnecting on disconnect.
// The OS (via launchd KeepAlive) owns the process lifecycle; this loop
// runs forever and never exits voluntarily.
func (p *program) run() {
	for {
		if err := p.client.Connect(); err != nil {
			slog.Warn("Could not connect to hub, retrying in 5s", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		slog.Info("Connected to hub")

		if err := p.client.Listen(); err != nil {
			slog.Warn("Disconnected from hub, reconnecting in 5s", "error", err)
			time.Sleep(5 * time.Second)
		}
		_ = p.client.Disconnect()
	}
}

// Stop is called by the service manager on SIGTERM.
// Lifecycle is managed by launchd (KeepAlive); nothing to clean up here.
func (p *program) Stop(s service.Service) error {
	return nil
}

// handleManifestRead reads a manifest file from user space and returns its raw
// content. The hub parses it and registers the node; use HYPHAE.file.hash to
// hash the manifest or its binary separately.
func (p *program) handleManifestRead(req *request.Request) {
	path, ok := req.Arg("path")
	if !ok {
		_ = p.client.SendResponseError(response.Error(req.Command(), req.Handle(), errArgumentMissing, "path"))
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		_ = p.client.SendResponseError(response.Error(req.Command(), req.Handle(), errRuntime, err.Error()))
		return
	}

	_ = p.client.SendResponse(response.New(req.Command(), req.Handle()).WithArg("content", string(data)))
}

// handleBinaryHash returns the SHA-256 hash of the binary for a running process
// identified by PID. The hub calls this at connect time to verify that the
// connecting node's binary matches what was recorded at install.
func (p *program) handleBinaryHash(req *request.Request) {
	pidStr, ok := req.Arg("pid")
	if !ok {
		_ = p.client.SendResponseError(response.Error(req.Command(), req.Handle(), errArgumentMissing, "pid"))
		return
	}
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		_ = p.client.SendResponseError(response.Error(req.Command(), req.Handle(), errArgumentInvalidType, "pid must be an integer"))
		return
	}

	binPath, err := binaryPathForPID(pid)
	if err != nil {
		_ = p.client.SendResponseError(response.Error(req.Command(), req.Handle(), errRuntime, err.Error()))
		return
	}

	hex, err := hashFile(binPath)
	if err != nil {
		_ = p.client.SendResponseError(response.Error(req.Command(), req.Handle(), errRuntime, err.Error()))
		return
	}

	_ = p.client.SendResponse(response.New(req.Command(), req.Handle()).WithArg("hash", hex))
}

// handleFileHash returns the SHA-256 hash of any file at the given path.
// Used at install time to hash a node binary or manifest without requiring
// a running process — the hub stores the digest in its registry.
func (p *program) handleFileHash(req *request.Request) {
	path, ok := req.Arg("path")
	if !ok {
		_ = p.client.SendResponseError(response.Error(req.Command(), req.Handle(), errArgumentMissing, "path"))
		return
	}

	hex, err := hashFile(path)
	if err != nil {
		_ = p.client.SendResponseError(response.Error(req.Command(), req.Handle(), errRuntime, err.Error()))
		return
	}

	_ = p.client.SendResponse(response.New(req.Command(), req.Handle()).WithArg("hash", hex))
}

// handleSpawn executes a binary in the user session. Fire-and-forget — the hub
// has already verified trust and learns the PID automatically via socket peer
// credentials when the spawned node connects.
func (p *program) handleSpawn(req *request.Request) {
	binary, ok := req.Arg("binary")
	if !ok {
		_ = p.client.SendResponseError(response.Error(req.Command(), req.Handle(), errArgumentMissing, "binary"))
		return
	}

	cmd := exec.Command(binary)
	if err := cmd.Start(); err != nil {
		_ = p.client.SendResponseError(response.Error(req.Command(), req.Handle(), errRuntime, err.Error()))
		return
	}
	// Detach: do not wait on the child. The hub tracks it via socket peercred.
	_ = p.client.SendResponse(response.New(req.Command(), req.Handle()))
}

// handleKill sends SIGTERM to the process with the given PID. If the process
// has not exited within 3 seconds it is force-killed with SIGKILL.
func (p *program) handleKill(req *request.Request) {
	pidStr, ok := req.Arg("pid")
	if !ok {
		_ = p.client.SendResponseError(response.Error(req.Command(), req.Handle(), errArgumentMissing, "pid"))
		return
	}
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		_ = p.client.SendResponseError(response.Error(req.Command(), req.Handle(), errArgumentInvalidType, "pid must be an integer"))
		return
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		_ = p.client.SendResponseError(response.Error(req.Command(), req.Handle(), errRuntime, err.Error()))
		return
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		_ = p.client.SendResponseError(response.Error(req.Command(), req.Handle(), errRuntime, err.Error()))
		return
	}

	// Wait up to 3 seconds for graceful exit, then force-kill.
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		if err := process.Signal(syscall.Signal(0)); err != nil {
			// Process is gone — clean exit after SIGTERM.
			_ = p.client.SendResponse(response.New(req.Command(), req.Handle()))
			return
		}
	}

	_ = process.Signal(syscall.SIGKILL)
	_ = p.client.SendResponse(response.New(req.Command(), req.Handle()))
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	prg := &program{}
	svc, err := service.New(prg, svcConfig)
	if err != nil {
		slog.Error("Failed to create service", "error", err)
		os.Exit(1)
	}

	// Handle service control subcommands:
	//   spore-agent install   — register with the OS service manager
	//   spore-agent uninstall — deregister
	//   spore-agent start     — start the background agent
	//   spore-agent stop      — stop the background agent
	//   spore-agent restart   — restart the background agent
	if len(os.Args) > 1 {
		// On macOS, kardianos/service uses the deprecated `launchctl load/unload`
		// API which fails on macOS Ventura+. Intercept start/stop/restart and use
		// the modern `launchctl bootstrap/bootout` commands for the gui/<uid> domain.
		if runtime.GOOS == "darwin" {
			home, err := os.UserHomeDir()
			if err != nil {
				slog.Error("Could not determine home directory", "error", err)
				os.Exit(1)
			}
			plist := filepath.Join(home, "Library", "LaunchAgents", svcConfig.Name+".plist")
			domain := "gui/" + strconv.Itoa(os.Getuid())

			switch os.Args[1] {
			case "start":
				if err := darwinLaunchctl("bootstrap", domain, plist); err != nil {
					slog.Error("Service control failed", "action", "start", "error", err)
					os.Exit(1)
				}
				return
			case "stop":
				// bootout returns an error if not running; treat that as a warning.
				if err := darwinLaunchctl("bootout", domain, plist); err != nil {
					slog.Warn("Service stop may have failed (not running?)", "error", err)
				}
				return
			case "restart":
				// bootout sends SIGKILL before bootstrap can run. Schedule bootstrap
				// in a detached shell that outlives this process.
				bootstrapScript := "sleep 1 && launchctl bootstrap " + domain + " " + plist + " &>/dev/null"
				_ = exec.Command("/bin/sh", "-c", bootstrapScript+" &").Run()
				_ = darwinLaunchctl("bootout", domain, plist) // ignore not-running error
				return
			case "uninstall":
				// Ensure the agent is stopped before kardianos deletes the plist.
				_ = darwinLaunchctl("bootout", domain, plist)
				// fall through to kardianos to remove the plist file
			}
		}

		if err := service.Control(svc, os.Args[1]); err != nil {
			slog.Error("Service control failed", "action", os.Args[1], "error", err)
			os.Exit(1)
		}
		return
	}

	// No subcommand: run directly (blocks until signal).
	if err := svc.Run(); err != nil {
		slog.Error("Service error", "error", err)
		os.Exit(1)
	}
}

// hashFile returns the SHA-256 hex digest of the file at path.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return "sha256:" + fmt.Sprintf("%x", h.Sum(nil)), nil
}

// binaryPathForPID returns the absolute path of the executable for the given
// PID using platform-specific APIs.
func binaryPathForPID(pid int) (string, error) {
	switch runtime.GOOS {
	case "linux":
		return os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	case "darwin":
		buf := make([]byte, 4096)
		ret, _, errno := syscall.Syscall(syscall.SYS_PROC_INFO,
			uintptr(9), // PROC_PIDPATHINFO
			uintptr(pid),
			uintptr(unsafe.Pointer(&buf[0])),
		)
		if errno != 0 || ret == 0 {
			return "", fmt.Errorf("proc_pidpath failed for pid %d: %w", pid, errno)
		}
		n := ret
		for i := uintptr(0); i < ret; i++ {
			if buf[i] == 0 {
				n = i
				break
			}
		}
		return string(buf[:n]), nil
	default:
		return "", fmt.Errorf("binaryPathForPID: unsupported platform %s", runtime.GOOS)
	}
}

// darwinLaunchctl runs a launchctl command and wraps any error with its output.
func darwinLaunchctl(args ...string) error {
	cmd := exec.Command("launchctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl %v: %w\n%s", args, err, out)
	}
	return nil
}
