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

func TestLoadFindsEnvInAParentDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("BOUNCER_DOTENV_PARENT=from-parent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "server")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	t.Cleanup(func() { _ = os.Unsetenv("BOUNCER_DOTENV_PARENT") })
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("BOUNCER_DOTENV_PARENT"); got != "from-parent" {
		t.Fatalf("BOUNCER_DOTENV_PARENT=%q", got)
	}
}

func TestLoadResolvesSiteRootFromTheEnvFile(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("SITE_ROOT=../apps\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "server")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	t.Cleanup(func() { _ = os.Unsetenv("SITE_ROOT") })
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(filepath.Join(root, "../apps"))
	if err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("SITE_ROOT"); got != want {
		t.Fatalf("SITE_ROOT=%q, want %q", got, want)
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
