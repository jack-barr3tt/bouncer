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
	opened, err := Open(dir, "")
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
	opened, err := Open(dir, "")
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
	opened, err := Open(dir, "")
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
	opened, err := Open(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = opened.Apps()
	if err == nil || !strings.Contains(err.Error(), "source") {
		t.Fatalf("error: %v", err)
	}
}

func TestAppsYAMLRejectsABadPath(t *testing.T) {
	dir := t.TempDir()
	body := "apps:\n  - slug: hello\n    name: Hello\n    description: Hi\n    path: /hello/\n    icon: Hi\n"
	if err := os.WriteFile(filepath.Join(dir, "apps.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	opened, err := Open(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = opened.Apps()
	if err == nil || !strings.Contains(err.Error(), "path") {
		t.Fatalf("error: %v", err)
	}
}
