// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/kardianos/service"
	spore "github.com/sporeos-dev/spore-client-libs/go"
)

const appId = "dev.sporeos.agent"

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
	done   chan struct{}
}

// Start is called by the service manager on launch.
func (p *program) Start(s service.Service) error {
	p.done = make(chan struct{})
	p.client = spore.NewClient(appId)

	// Register handlers — called by the hub when it needs user-space access.
	p.client.HandleRequest("HYPHAE.node.install", p.handleInstall)
	p.client.HandleRequest("HYPHAE.node.uninstall", p.handleUninstall)
	p.client.HandleRequest("HYPHAE.node.spawn", p.handleSpawn)
	p.client.HandleRequest("HYPHAE.node.kill", p.handleKill)

	go p.run()
	return nil
}

// run connects to the hub and listens, reconnecting on disconnect.
func (p *program) run() {
	for {
		select {
		case <-p.done:
			return
		default:
		}

		if err := p.client.Connect(); err != nil {
			slog.Warn("Could not connect to hub, retrying in 5s", "error", err)
			select {
			case <-p.done:
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}

		slog.Info("Connected to hub")

		if err := p.client.Listen(); err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			slog.Warn("Disconnected from hub, reconnecting in 5s", "error", err)
		}
		p.client.Close()

		select {
		case <-p.done:
			return
		case <-time.After(5 * time.Second):
		}
	}
}

// Stop is called by the service manager on SIGTERM or `spore-agent stop`.
func (p *program) Stop(s service.Service) error {
	slog.Info("Spore Agent stopping")
	close(p.done)
	p.client.Close()
	return nil
}

// handleInstall is called by the hub when it needs to install a node whose
// manifest lives in the user's file space (e.g. ~/Documents/dev/).
func (p *program) handleInstall(call *spore.Call) {
	if !call.HasArg("path") {
		_ = call.Error(spore.ErrorCodeArgumentMissing, "path")
		return
	}
	path := call.Arg("path")
	// TODO: validate the manifest is readable and well-formed, then hand
	// back to the hub (e.g. confirm the path and return manifest metadata).
	_ = path
	_ = call.Reply(nil)
}

// handleUninstall is called by the hub to uninstall a node from user space.
func (p *program) handleUninstall(call *spore.Call) {
	if !call.HasArg("node") {
		_ = call.Error(spore.ErrorCodeArgumentMissing, "node")
		return
	}
	nodeID := call.Arg("node")
	// TODO: clean up any user-space artefacts for this node.
	_ = nodeID
	_ = call.Reply(nil)
}

// handleSpawn is called by the hub to start a node process in user space.
// Most nodes run as the logged-in user; the hub delegates the exec here so
// the process inherits the user session rather than the _spore account.
func (p *program) handleSpawn(call *spore.Call) {
	if !call.HasArg("node") {
		_ = call.Error(spore.ErrorCodeArgumentMissing, "node")
		return
	}
	nodeID := call.Arg("node")
	// TODO: resolve the node binary path from its manifest and exec it.
	_ = nodeID
	_ = call.Reply(nil)
}

// handleKill is called by the hub to stop a node process running in user space.
func (p *program) handleKill(call *spore.Call) {
	if !call.HasArg("node") {
		_ = call.Error(spore.ErrorCodeArgumentMissing, "node")
		return
	}
	nodeID := call.Arg("node")
	// TODO: find the process for this node and terminate it.
	_ = nodeID
	_ = call.Reply(nil)
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

// darwinLaunchctl runs a launchctl command and wraps any error with its output.
func darwinLaunchctl(args ...string) error {
	cmd := exec.Command("launchctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl %v: %w\n%s", args, err, out)
	}
	return nil
}
