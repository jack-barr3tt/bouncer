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
	siteRoot := t.TempDir()
	writeFile(t, filepath.Join(siteRoot, "apps.yaml"), sourceRegistry())
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
	gamesRemote := gamesOrigin
	notesRemote := notesOrigin
	yaml := sourceRegistryRemotes(notesRemote, gamesRemote)
	writeFile(t, filepath.Join(siteRoot, "apps.yaml"), yaml)
	writeFile(t, filepath.Join(siteRoot, "apps", "notes", "dist", "index.html"), "old-notes")
	writeFile(t, filepath.Join(siteRoot, "apps", "chess", "dist", "index.html"), "old-chess")
	writeFile(t, filepath.Join(siteRoot, "apps", "draft", "dist", "index.html"), "old-draft")
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

	chess, err := os.ReadFile(filepath.Join(siteRoot, "apps", "chess", "dist", "index.html"))
	if err != nil || string(chess) != "built-chess" {
		t.Fatalf("chess: %q %v", chess, err)
	}
	draft, err := os.ReadFile(filepath.Join(siteRoot, "apps", "draft", "dist", "index.html"))
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
	if len(built) != 1 || !strings.Contains(built[0], "chess") || !strings.Contains(built[0], "src/repos/games/chess") {
		t.Fatalf("built %v", built)
	}
	latest, err := svc.store.Latest(context.Background(), gamesRemote)
	if err != nil || latest == nil || latest.Status != "published" || latest.SHA != second {
		t.Fatalf("latest: %+v %v", latest, err)
	}

	if err := os.RemoveAll(filepath.Join(siteRoot, "apps", "chess", "dist")); err != nil {
		t.Fatal(err)
	}
	built = nil
	svc.perform(context.Background(), gamesRemote, second)
	chess, err = os.ReadFile(filepath.Join(siteRoot, "apps", "chess", "dist", "index.html"))
	if err != nil || string(chess) != "built-chess" {
		t.Fatalf("rebuilt chess: %q %v", chess, err)
	}
	if len(built) != 1 || !strings.Contains(built[0], "chess") {
		t.Fatalf("rebuilt %v", built)
	}
}

func TestPinnedRepoIgnoresPush(t *testing.T) {
	siteRoot := t.TempDir()
	pinned := "0123456789abcdef0123456789abcdef01234567"
	writeFile(t, filepath.Join(siteRoot, "apps.yaml"), ""+
		"repos:\n"+
		"  - name: notes\n    remote: git@github.com:you/notes.git\n    commit: "+pinned+"\n"+
		"apps:\n"+
		"  - slug: notes\n    name: Notes\n    description: Hi\n    path: /apps/notes/\n    icon: N\n    repo: notes\n")
	svc := sourceService(t, siteRoot)
	body := []byte(`{"ref":"refs/heads/main","after":"abcdef0123456789abcdef0123456789abcdef01","repository":{"clone_url":"https://github.com/you/notes.git"}}`)
	status, remote, sha := svc.DecideHook("push", sign("secret", body), body)
	if status != http.StatusNoContent || remote != "" || sha != "" {
		t.Fatalf("pinned: %d %s %s", status, remote, sha)
	}
	svc.perform(context.Background(), "git@github.com:you/notes.git", "abcdef0123456789abcdef0123456789abcdef01")
	latest, err := svc.store.Latest(context.Background(), "git@github.com:you/notes.git")
	if err != nil || latest == nil || latest.Status != "failed" || !strings.Contains(latest.Log, "pinned") {
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
	return sourceRegistryRemotes("git@github.com:you/notes.git", "https://github.com/you/games.git")
}

func sourceRegistryRemotes(notes, games string) string {
	return "repos:\n" +
		"  - name: notes\n    remote: " + notes + "\n    branch: main\n" +
		"  - name: games\n    remote: " + games + "\n    branch: main\n" +
		"apps:\n" +
		"  - slug: notes\n    name: Notes\n    description: Hi\n    path: /apps/notes/\n    icon: N\n    repo: notes\n" +
		"  - slug: chess\n    name: Chess\n    description: Hi\n    path: /apps/chess/\n    icon: C\n    repo: games\n    dir: chess\n" +
		"  - slug: draft\n    name: Draft\n    description: Hi\n    path: /apps/draft/\n    icon: D\n    repo: games\n    dir: draft\n"
}
