// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"errors"
	"testing"

	"github.com/sporeos-dev/spore-client-libs/spore_go/request"
)

func TestConfirmReturnsYesAfterAuthentication(t *testing.T) {
	req := request.New("HYPHAE.confirm", "h1").
		WithArg("node", "dev.example.mcp").
		WithArg("command", "node.install")

	res, resErr := confirm(req, func(reason string) (bool, error) {
		want := "Spore wants to authorize dev.example.mcp to run node.install."
		if reason != want {
			t.Fatalf("reason = %q, want %q", reason, want)
		}
		return true, nil
	})

	if resErr != nil {
		t.Fatalf("confirm returned error: %s", resErr.What())
	}
	if !res.Flag("yes") || res.Flag("no") {
		t.Fatalf("response flags must contain only yes")
	}
}

func TestConfirmReturnsNoWhenAuthenticationIsDenied(t *testing.T) {
	req := request.New("HYPHAE.confirm", "h1").
		WithArg("node", "dev.example.mcp").
		WithArg("command", "node.install")

	res, resErr := confirm(req, func(string) (bool, error) {
		return false, nil
	})

	if resErr != nil {
		t.Fatalf("confirm returned error: %s", resErr.What())
	}
	if !res.Flag("no") || res.Flag("yes") {
		t.Fatalf("response flags must contain only no")
	}
}

func TestConfirmRejectsMissingIdentityAndAuthenticationFailures(t *testing.T) {
	missingNode := request.New("HYPHAE.confirm", "h1").WithArg("command", "node.install")
	if _, resErr := confirm(missingNode, func(string) (bool, error) { return true, nil }); resErr == nil ||
		resErr.Code() != errArgumentMissing {
		t.Fatalf("missing node error = %#v, want %s", resErr, errArgumentMissing)
	}

	req := request.New("HYPHAE.confirm", "h1").
		WithArg("node", "dev.example.mcp").
		WithArg("command", "node.install")
	if _, resErr := confirm(req, func(string) (bool, error) {
		return false, errors.New("authentication unavailable")
	}); resErr == nil || resErr.Code() != errRuntime {
		t.Fatalf("authentication error = %#v, want %s", resErr, errRuntime)
	}
}

func TestUnsupportedAuthenticationReturnsAnError(t *testing.T) {
	confirmed, err := unsupportedAuthentication("Spore wants to authorize a node.")
	if confirmed {
		t.Fatal("unsupported authentication unexpectedly confirmed")
	}
	if err == nil {
		t.Fatal("unsupported authentication returned no error")
	}
}
