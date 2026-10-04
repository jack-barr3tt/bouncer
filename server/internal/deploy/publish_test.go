package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPublishSwapsDistAndDropsRemovedApps(t *testing.T) {
	siteRoot := t.TempDir()
	checkout := t.TempDir()
	output := t.TempDir()
	writeFile(t, filepath.Join(siteRoot, "apps.yaml"), registry("hello", "notes"))
	writeFile(t, filepath.Join(siteRoot, "apps", "hello", "dist", "index.html"), "old")
	writeFile(t, filepath.Join(siteRoot, "apps", "notes", "dist", "index.html"), "notes")
	writeFile(t, filepath.Join(checkout, "apps.yaml"), registry("hello"))
	writeFile(t, filepath.Join(output, "apps", "hello", "dist", "index.html"), "new")

	if err := Publish(siteRoot, checkout, output, []string{"hello"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(siteRoot, "apps", "hello", "dist", "index.html"))
	if err != nil || string(body) != "new" {
		t.Fatalf("dist: %q %v", body, err)
	}
	if _, err := os.Stat(filepath.Join(siteRoot, "apps", "hello", "dist.next")); !os.IsNotExist(err) {
		t.Fatal("dist.next still present")
	}
	if _, err := os.Stat(filepath.Join(siteRoot, "apps", "hello", "dist.prev")); !os.IsNotExist(err) {
		t.Fatal("dist.prev still present")
	}
	if _, err := os.Stat(filepath.Join(siteRoot, "apps", "notes")); !os.IsNotExist(err) {
		t.Fatal("removed app still present")
	}
	yaml, err := os.ReadFile(filepath.Join(siteRoot, "apps.yaml"))
	if err != nil || string(yaml) != registry("hello") {
		t.Fatalf("yaml: %q %v", yaml, err)
	}
}

func TestPublishRejectsSymlinkWithoutReplacingDist(t *testing.T) {
	siteRoot := t.TempDir()
	checkout := t.TempDir()
	output := t.TempDir()
	writeFile(t, filepath.Join(siteRoot, "apps.yaml"), registry("hello"))
	writeFile(t, filepath.Join(siteRoot, "apps", "hello", "dist", "index.html"), "old")
	writeFile(t, filepath.Join(checkout, "apps.yaml"), registry("hello"))
	dist := filepath.Join(output, "apps", "hello", "dist")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(siteRoot, "apps.yaml"), filepath.Join(dist, "link")); err != nil {
		t.Fatal(err)
	}
	if err := Publish(siteRoot, checkout, output, []string{"hello"}); err == nil {
		t.Fatal("expected symlink to fail publish")
	}
	body, err := os.ReadFile(filepath.Join(siteRoot, "apps", "hello", "dist", "index.html"))
	if err != nil || string(body) != "old" {
		t.Fatalf("live dist: %q %v", body, err)
	}
}

func registry(slugs ...string) string {
	body := "apps:\n"
	for _, slug := range slugs {
		body += "  - slug: " + slug + "\n    name: " + slug + "\n    description: Hi\n    path: /apps/" + slug + "/\n    icon: Hi\n"
	}
	return body
}
