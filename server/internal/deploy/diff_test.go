package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldBuildAll(t *testing.T) {
	if !ShouldBuildAll("") || !ShouldBuildAll("0000000") {
		t.Fatal("missing base should build every app")
	}
	if ShouldBuildAll("abc123") {
		t.Fatal("a real base should not build every app")
	}
}

func TestSlugsFromDiff(t *testing.T) {
	diff := "apps/hello/src/App.tsx\napps/hub/index.html\napps.yaml\nREADME.md\n"
	got := SlugsFromDiff(diff, []string{"hello", "notes"})
	if len(got) != 1 || got[0] != "hello" {
		t.Fatalf("slugs: %v", got)
	}
}

func TestAppsFromDiff(t *testing.T) {
	diff := "apps/chess/src/App.tsx\nREADME.md\n"
	got := AppsFromDiff(diff, []sourcedApp{{Slug: "chess", Rel: "apps/chess"}, {Slug: "notes", Rel: "apps/notes"}, {Slug: "root", Rel: "."}})
	if len(got) != 2 || got[0] != "chess" || got[1] != "root" {
		t.Fatalf("slugs: %v", got)
	}
}

func TestListAppSlugs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "hello", "package.json"), "{}")
	writeFile(t, filepath.Join(dir, "hub", "package.json"), "{}")
	if err := os.Mkdir(filepath.Join(dir, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := ListAppSlugs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "hello" {
		t.Fatalf("slugs: %v", got)
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
