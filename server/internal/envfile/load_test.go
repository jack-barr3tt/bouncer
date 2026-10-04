package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := Load(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadSetsUnsetVariables(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("BOUNCER_DOTENV_NEW=from-file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Cleanup(func() { _ = os.Unsetenv("BOUNCER_DOTENV_NEW") })
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("BOUNCER_DOTENV_NEW"); got != "from-file" {
		t.Fatalf("BOUNCER_DOTENV_NEW=%q", got)
	}
}

func TestLoadKeepsExistingVariables(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("BOUNCER_DOTENV_KEEP=from-file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BOUNCER_DOTENV_KEEP", "from-env")
	t.Chdir(dir)
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("BOUNCER_DOTENV_KEEP"); got != "from-env" {
		t.Fatalf("BOUNCER_DOTENV_KEEP=%q", got)
	}
}
