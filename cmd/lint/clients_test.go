package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveClientsDefaultFileInCWD(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(".winze-clients.json", []byte(`{"cwd-client": "/checkout/cwd"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := resolveClients("", "")
	if err != nil {
		t.Fatal(err)
	}
	if c["cwd-client"] != "/checkout/cwd" {
		t.Errorf("got %+v", c)
	}
}

func TestResolveClientsEmptyWhenNothingConfigured(t *testing.T) {
	t.Chdir(t.TempDir())
	c, err := resolveClients("", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(c) != 0 {
		t.Errorf("expected an empty map, got %+v", c)
	}
}

func TestResolveClientsEnvVarFallback(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "clients.json")
	if err := os.WriteFile(file, []byte(`{"env-client": "/checkout/env"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WINZE_CLIENTS_FILE", file)
	c, err := resolveClients("", "")
	if err != nil {
		t.Fatal(err)
	}
	if c["env-client"] != "/checkout/env" {
		t.Errorf("got %+v", c)
	}
}

func TestResolveClientsFlagWinsOverFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "clients.json")
	if err := os.WriteFile(file, []byte(`{"a": "/from-file"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := resolveClients("a=/from-flag", file)
	if err != nil {
		t.Fatal(err)
	}
	if c["a"] != "/from-flag" {
		t.Errorf("flag should win over file, got %q", c["a"])
	}
}

func TestResolveClientsFromFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "clients.json")
	if err := os.WriteFile(file, []byte(`{"wovim": "/checkout/wovim"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := resolveClients("", file)
	if err != nil {
		t.Fatal(err)
	}
	if c["wovim"] != "/checkout/wovim" {
		t.Errorf("got %+v", c)
	}
}

func TestResolveClientsFromFlag(t *testing.T) {
	c, err := resolveClients("a=/path/a,b=/path/b", "")
	if err != nil {
		t.Fatal(err)
	}
	if c["a"] != "/path/a" || c["b"] != "/path/b" {
		t.Errorf("got %+v", c)
	}
}

func TestResolveClientsMalformedPairErrors(t *testing.T) {
	if _, err := resolveClients("noequals", ""); err == nil {
		t.Error("expected an error for a pair with no '='")
	}
	if _, err := resolveClients("=novalue", ""); err == nil {
		t.Error("expected an error for an empty name")
	}
	if _, err := resolveClients("noname=", ""); err == nil {
		t.Error("expected an error for an empty path")
	}
}
