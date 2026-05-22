package dotenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBasic(t *testing.T) {
	dir := t.TempDir()
	body := "FOO=bar\n# a comment line\nBAZ=qux\n\nKEY_WITH_SPACES = trimmed \n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("FOO", "")
	t.Setenv("BAZ", "")
	t.Setenv("KEY_WITH_SPACES", "")

	Load(dir)

	if got := os.Getenv("FOO"); got != "bar" {
		t.Errorf("FOO=%q, want bar", got)
	}
	if got := os.Getenv("BAZ"); got != "qux" {
		t.Errorf("BAZ=%q, want qux", got)
	}
	if got := os.Getenv("KEY_WITH_SPACES"); got != "trimmed" {
		t.Errorf("KEY_WITH_SPACES=%q, want trimmed", got)
	}
}

func TestLoadDoesNotOverridePresent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("ALREADY_SET=from-env-file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ALREADY_SET", "from-process")

	Load(dir)

	if got := os.Getenv("ALREADY_SET"); got != "from-process" {
		t.Errorf("ALREADY_SET=%q, want from-process (Load must not override)", got)
	}
}

func TestLoadMissingFileIsSilent(t *testing.T) {
	dir := t.TempDir() // no .env present
	Load(dir)          // must not panic, must not error visibly
}

func TestLoadMalformedLinesSkipped(t *testing.T) {
	dir := t.TempDir()
	body := "no-equals-sign\nVALID=ok\nanother bad line\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VALID", "")
	Load(dir)
	if got := os.Getenv("VALID"); got != "ok" {
		t.Errorf("VALID=%q, want ok", got)
	}
}
