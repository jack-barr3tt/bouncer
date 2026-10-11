package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppsYAML(t *testing.T) {
	dir := t.TempDir()
	body := "apps:\n  - slug: hello\n    name: Hello\n    description: Hi\n    path: /apps/hello/\n    icon: Hi\n"
	if err := os.WriteFile(filepath.Join(dir, "apps.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	opened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	apps, err := opened.Apps()
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0].Slug != "hello" || apps[0].Path != "/apps/hello/" || apps[0].Source != "" {
		t.Fatalf("apps: %+v", apps)
	}
}

func TestAppsYAMLKeepsSource(t *testing.T) {
	dir := t.TempDir()
	body := "apps:\n  - slug: notes\n    name: Notes\n    description: Hi\n    path: /apps/notes/\n    icon: N\n    source: apps/games/notes/\n"
	if err := os.WriteFile(filepath.Join(dir, "apps.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	opened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	apps, err := opened.Apps()
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0].Source != "apps/games/notes" {
		t.Fatalf("apps: %+v", apps)
	}
}

func TestAppsYAMLRejectsASourceOutsideApps(t *testing.T) {
	dir := t.TempDir()
	body := "apps:\n  - slug: hello\n    name: Hello\n    description: Hi\n    path: /apps/hello/\n    icon: Hi\n    source: notes\n"
	if err := os.WriteFile(filepath.Join(dir, "apps.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	opened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = opened.Apps()
	if err == nil || !strings.Contains(err.Error(), "apps/") {
		t.Fatalf("error: %v", err)
	}
}

func TestAppsYAMLRejectsASourceThatEscapes(t *testing.T) {
	dir := t.TempDir()
	body := "apps:\n  - slug: hello\n    name: Hello\n    description: Hi\n    path: /apps/hello/\n    icon: Hi\n    source: ../secret\n"
	if err := os.WriteFile(filepath.Join(dir, "apps.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	opened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = opened.Apps()
	if err == nil || !strings.Contains(err.Error(), "source") {
		t.Fatalf("error: %v", err)
	}
}

func TestAppsYAMLKeepsUpstream(t *testing.T) {
	dir := t.TempDir()
	body := "apps:\n  - slug: notes\n    name: Notes\n    description: Hi\n    path: /apps/notes/\n    icon: N\n    upstream: http://127.0.0.1:3000/\n"
	if err := os.WriteFile(filepath.Join(dir, "apps.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	opened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	apps, err := opened.Apps()
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0].Upstream != "http://127.0.0.1:3000" || apps[0].Source != "" {
		t.Fatalf("apps: %+v", apps)
	}
}

func TestAppsYAMLRejectsUpstreamWithSource(t *testing.T) {
	rejectUpstream(t, "http://127.0.0.1:9", "apps/notes", "both")
}

func TestAppsYAMLRejectsUpstreamUserinfo(t *testing.T) {
	rejectUpstream(t, "http://user:secret@127.0.0.1:9", "", "username")
}

func TestAppsYAMLRejectsUpstreamPath(t *testing.T) {
	rejectUpstream(t, "http://127.0.0.1:9/notes", "", "path")
}

func rejectUpstream(t *testing.T, upstream, source, want string) {
	t.Helper()
	dir := t.TempDir()
	body := "apps:\n  - slug: notes\n    name: Notes\n    description: Hi\n    path: /apps/notes/\n    icon: N\n    upstream: " + upstream + "\n"
	if source != "" {
		body += "    source: " + source + "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "apps.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	opened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = opened.Apps()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error: %v", err)
	}
}

func TestAppsYAMLKeepsAPinnedRepo(t *testing.T) {
	dir := t.TempDir()
	body := "repos:\n  - name: games\n    remote: git@github.com:you/games.git\n    commit: abcdef0123456789abcdef0123456789abcdef01\napps:\n  - slug: chess\n    name: Chess\n    description: Hi\n    path: /apps/chess/\n    icon: C\n    repo: games\n    dir: chess/\n"
	if err := os.WriteFile(filepath.Join(dir, "apps.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	opened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	file, err := opened.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Repos) != 1 || file.Repos[0].Commit == "" || file.Repos[0].Branch != "" {
		t.Fatalf("repos: %+v", file.Repos)
	}
	if len(file.Apps) != 1 || file.Apps[0].Repo != "games" || file.Apps[0].Dir != "chess" {
		t.Fatalf("apps: %+v", file.Apps)
	}
	got := opened.AppDir("chess")
	if got != filepath.Join(opened.Root(), "apps", "chess", "dist") {
		t.Fatalf("dir: %s", got)
	}
}

func TestAppsYAMLRejectsRepoWithSource(t *testing.T) {
	dir := t.TempDir()
	body := "repos:\n  - name: games\n    remote: git@github.com:you/games.git\n    branch: main\napps:\n  - slug: chess\n    name: Chess\n    description: Hi\n    path: /apps/chess/\n    icon: C\n    repo: games\n    source: apps/chess\n"
	if err := os.WriteFile(filepath.Join(dir, "apps.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	opened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = opened.Load()
	if err == nil || !strings.Contains(err.Error(), "repo") {
		t.Fatalf("error: %v", err)
	}
}

func TestAppsYAMLRejectsCommitAndBranch(t *testing.T) {
	dir := t.TempDir()
	body := "repos:\n  - name: games\n    remote: git@github.com:you/games.git\n    commit: abcdef0123456789abcdef0123456789abcdef01\n    branch: main\napps: []\n"
	if err := os.WriteFile(filepath.Join(dir, "apps.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	opened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = opened.Load()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAppsYAMLRejectsABadPath(t *testing.T) {
	dir := t.TempDir()
	body := "apps:\n  - slug: hello\n    name: Hello\n    description: Hi\n    path: /hello/\n    icon: Hi\n"
	if err := os.WriteFile(filepath.Join(dir, "apps.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	opened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = opened.Apps()
	if err == nil || !strings.Contains(err.Error(), "path") {
		t.Fatalf("error: %v", err)
	}
}
