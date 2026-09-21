package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCmd_Run_PrintsGreeting(t *testing.T) {
	cmd := newRootCmd()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned an error: %v", err)
	}

	if !strings.Contains(out.String(), "Lets Go!") {
		t.Errorf("expected greeting on stdout, got stdout=%q stderr=%q", out.String(), errOut.String())
	}

	if errOut.String() != "" {
		t.Errorf("expected no stderr output, got %q", errOut.String())
	}
}

func TestRootCmd_Version(t *testing.T) {
	Version = "v1.2.3"
	Build = "abc123"

	t.Cleanup(func() {
		Version = ""
		Build = ""
	})

	cmd := newRootCmd()

	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned an error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, Version) || !strings.Contains(got, Build) {
		t.Errorf("expected version output to contain %q and %q, got %q", Version, Build, got)
	}
}

func TestRootCmd_CompletionZsh(t *testing.T) {
	cmd := newRootCmd()

	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"completion", "zsh"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned an error: %v", err)
	}

	if !strings.HasPrefix(out.String(), "#compdef") {
		t.Errorf("expected zsh completion script to start with #compdef, got %q", out.String()[:min(40, len(out.String()))])
	}
}
