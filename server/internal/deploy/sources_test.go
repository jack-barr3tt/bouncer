package deploy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSameRemote(t *testing.T) {
	if !sameRemote("git@github.com:you/notes.git", "https://github.com/you/notes") {
		t.Fatal("ssh and https should match")
	}
	if !sameRemote("ssh://git@github.com/you/notes.git", "https://github.com/you/notes.git") {
		t.Fatal("ssh url and https should match")
	}
	if sameRemote("git@github.com:you/notes.git", "https://github.com/you/games.git") {
		t.Fatal("different repos matched")
	}
}

func TestHookMatchesTheClone(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	siteRoot := t.TempDir()
	initRepo(t, filepath.Join(siteRoot, "apps", "notes"), map[string]string{"package.json": "{}"})
	initRepo(t, filepath.Join(siteRoot, "apps", "games"), map[string]string{"chess/package.json": "{}"})
	gitCmd(t, filepath.Join(siteRoot, "apps", "notes"), "remote", "add", "origin", "git@github.com:you/notes.git")
	gitCmd(t, filepath.Join(siteRoot, "apps", "games"), "remote", "add", "origin", "https://github.com/you/games.git")
	writeFile(t, filepath.Join(siteRoot, "apps.yaml"), ""+
		"apps:\n"+
		"  - slug: notes\n    name: Notes\n    description: Hi\n    path: /apps/notes/\n    icon: N\n    source: apps/notes\n"+
		"  - slug: chess\n    name: Chess\n    description: Hi\n    path: /apps/chess/\n    icon: C\n    source: apps/games/chess\n")
	svc := sourceService(t, siteRoot)

	body := []byte(`{"ref":"refs/heads/main","after":"0123456789abcdef0123456789abcdef01234567","repository":{"clone_url":"https://github.com/you/notes.git"}}`)
	status, remote, sha := svc.DecideHook("push", sign("secret", body), body)
	if status != http.StatusAccepted || remote != "git@github.com:you/notes.git" || sha != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("notes: %d %s %s", status, remote, sha)
	}
	other := []byte(`{"ref":"refs/heads/main","after":"0123456789abcdef0123456789abcdef01234567","repository":{"ssh_url":"git@github.com:you/other.git"}}`)
	status, remote, sha = svc.DecideHook("push", sign("secret", other), other)
	if status != http.StatusNoContent || remote != "" || sha != "" {
		t.Fatalf("unknown: %d %s %s", status, remote, sha)
	}
	target, err := svc.Target("")
	if err == nil || target != "" {
		t.Fatalf("target: %q %v", target, err)
	}
	target, err = svc.Target("https://github.com/you/games")
	if err != nil || target != "https://github.com/you/games.git" {
		t.Fatalf("games: %q %v", target, err)
	}
}

func TestPublishSourceApps(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	notesOrigin := t.TempDir()
	gamesOrigin := t.TempDir()
	initRepo(t, notesOrigin, map[string]string{"package.json": "{}", "index.js": "one"})
	initRepo(t, gamesOrigin, map[string]string{
		"chess/package.json": "{}",
		"chess/index.js":     "one",
		"draft/package.json": "{}",
		"draft/index.js":     "one",
	})
	first := strings.TrimSpace(gitCmd(t, gamesOrigin, "rev-parse", "HEAD"))
	writeFile(t, filepath.Join(gamesOrigin, "chess", "index.js"), "two")
	gitCmd(t, gamesOrigin, "add", ".")
	gitCmd(t, gamesOrigin, "commit", "-m", "second")
	second := strings.TrimSpace(gitCmd(t, gamesOrigin, "rev-parse", "HEAD"))

	siteRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(siteRoot, "apps"), 0o755); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, siteRoot, "clone", notesOrigin, "apps/notes")
	gitCmd(t, siteRoot, "clone", gamesOrigin, "apps/games")
	yaml := sourceRegistry()
	writeFile(t, filepath.Join(siteRoot, "apps.yaml"), yaml)
	writeFile(t, filepath.Join(siteRoot, "apps", "notes", "dist", "index.html"), "old-notes")
	writeFile(t, filepath.Join(siteRoot, "apps", "games", "chess", "dist", "index.html"), "old-chess")
	writeFile(t, filepath.Join(siteRoot, "apps", "games", "draft", "dist", "index.html"), "old-draft")

	gamesRemote := strings.TrimSpace(gitCmd(t, filepath.Join(siteRoot, "apps", "games"), "remote", "get-url", "origin"))
	var built []string
	workspace := t.TempDir()
	builder := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Apps []BuildApp `json:"apps"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad", http.StatusBadRequest)
			return
		}
		for _, app := range req.Apps {
			built = append(built, app.Slug+":"+app.Dir)
			writeFile(t, filepath.Join(workspace, "out", "apps", app.Slug, "dist", "index.html"), "built-"+app.Slug)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "log": "built\n"})
	}))
	defer builder.Close()

	svc := sourceService(t, siteRoot)
	svc.cfg.BuilderURL = builder.URL
	svc.cfg.Workspace = workspace
	svc.store.(*memStore).seed(gamesRemote, first)
	svc.perform(context.Background(), gamesRemote, second)

	chess, err := os.ReadFile(filepath.Join(siteRoot, "apps", "games", "chess", "dist", "index.html"))
	if err != nil || string(chess) != "built-chess" {
		t.Fatalf("chess: %q %v", chess, err)
	}
	draft, err := os.ReadFile(filepath.Join(siteRoot, "apps", "games", "draft", "dist", "index.html"))
	if err != nil || string(draft) != "old-draft" {
		t.Fatalf("draft: %q %v", draft, err)
	}
	notes, err := os.ReadFile(filepath.Join(siteRoot, "apps", "notes", "dist", "index.html"))
	if err != nil || string(notes) != "old-notes" {
		t.Fatalf("notes: %q %v", notes, err)
	}
	gotYAML, err := os.ReadFile(filepath.Join(siteRoot, "apps.yaml"))
	if err != nil || string(gotYAML) != yaml {
		t.Fatalf("yaml changed: %q %v", gotYAML, err)
	}
	if len(built) != 1 || !strings.Contains(built[0], "chess") || !strings.Contains(built[0], "src/apps/games/chess") {
		t.Fatalf("built %v", built)
	}
	latest, err := svc.store.Latest(context.Background(), gamesRemote)
	if err != nil || latest == nil || latest.Status != "published" || latest.SHA != second {
		t.Fatalf("latest: %+v %v", latest, err)
	}

	os.RemoveAll(filepath.Join(siteRoot, "apps", "games", "chess", "dist"))
	built = nil
	svc.perform(context.Background(), gamesRemote, second)
	chess, err = os.ReadFile(filepath.Join(siteRoot, "apps", "games", "chess", "dist", "index.html"))
	if err != nil || string(chess) != "built-chess" {
		t.Fatalf("rebuilt chess: %q %v", chess, err)
	}
	if len(built) != 1 || !strings.Contains(built[0], "chess") {
		t.Fatalf("rebuilt %v", built)
	}
}

func TestDetachedCloneFails(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	siteRoot := t.TempDir()
	initRepo(t, filepath.Join(siteRoot, "apps", "notes"), map[string]string{"package.json": "{}"})
	gitCmd(t, filepath.Join(siteRoot, "apps", "notes"), "remote", "add", "origin", "git@github.com:you/notes.git")
	gitCmd(t, filepath.Join(siteRoot, "apps", "notes"), "checkout", "--detach")
	writeFile(t, filepath.Join(siteRoot, "apps.yaml"), "apps:\n  - slug: notes\n    name: Notes\n    description: Hi\n    path: /apps/notes/\n    icon: N\n    source: apps/notes\n")
	svc := sourceService(t, siteRoot)
	svc.perform(context.Background(), "git@github.com:you/notes.git", "")
	latest, err := svc.store.Latest(context.Background(), "git@github.com:you/notes.git")
	if err != nil || latest == nil || latest.Status != "failed" || !strings.Contains(latest.Log, "detached HEAD") {
		t.Fatalf("latest: %+v %v", latest, err)
	}
}

func sourceService(t *testing.T, siteRoot string) *Service {
	t.Helper()
	svc, err := New(Config{
		WebhookSecret: "secret",
		DeployToken:   "token",
		BuilderToken:  "builder",
		BuilderURL:    "http://127.0.0.1:1",
		Workspace:     t.TempDir(),
		SiteRoot:      siteRoot,
		PublicBase:    "https://apps.example.com",
		RuntimeDir:    t.TempDir(),
		KeyFile:       filepath.Join(t.TempDir(), "missing-key"),
		store:         &memStore{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func initRepo(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, dir, "init", "-b", "main")
	for path, body := range files {
		writeFile(t, filepath.Join(dir, path), body)
	}
	gitCmd(t, dir, "add", ".")
	gitCmd(t, dir, "commit", "-m", "first")
}

func sourceRegistry() string {
	return "apps:\n" +
		"  - slug: notes\n    name: Notes\n    description: Hi\n    path: /apps/notes/\n    icon: N\n    source: apps/notes\n" +
		"  - slug: chess\n    name: Chess\n    description: Hi\n    path: /apps/chess/\n    icon: C\n    source: apps/games/chess\n" +
		"  - slug: draft\n    name: Draft\n    description: Hi\n    path: /apps/draft/\n    icon: D\n    source: apps/games/draft\n"
}
