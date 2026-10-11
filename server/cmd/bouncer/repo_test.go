package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jack-barr3tt/bouncer/internal/site"
)

func TestRepoAddWritesReference(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	origin := t.TempDir()
	gitInit(t, origin, map[string]string{"package.json": "{}", "chess/package.json": "{}"})
	siteRoot := t.TempDir()
	write(t, filepath.Join(siteRoot, "apps.yaml"), "apps:\n  - slug: hello\n    name: Hello\n    description: Hi\n    path: /apps/hello/\n    icon: H\n")
	git(t, siteRoot, "init")
	var out bytes.Buffer
	err := runRepo([]string{"add", origin, "--site", siteRoot, "--name", "games", "--branch", "main", "--app", ".=notes", "--app", "chess=chess"}, strings.NewReader(""), &out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Commit apps.yaml") {
		t.Fatalf("output: %s", out.String())
	}
	opened, err := site.Open(siteRoot)
	if err != nil {
		t.Fatal(err)
	}
	file, err := opened.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Repos) != 1 || file.Repos[0].Name != "games" || file.Repos[0].Branch != "main" || file.Repos[0].Commit != "" {
		t.Fatalf("repos: %+v", file.Repos)
	}
	if len(file.Apps) != 3 {
		t.Fatalf("apps: %+v", file.Apps)
	}
	if _, err := os.Stat(filepath.Join(siteRoot, "apps", "games")); !os.IsNotExist(err) {
		t.Fatalf("clone left in the site: %v", err)
	}
}

func gitInit(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	git(t, dir, "init", "-b", "main")
	for path, body := range files {
		write(t, filepath.Join(dir, path), body)
	}
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-m", "first")
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	base := []string{"-c", "commit.gpgsign=false", "-c", "user.email=test@example.com", "-c", "user.name=test"}
	cmd := exec.Command("git", append(base, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}
