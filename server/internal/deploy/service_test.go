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
	"sync"
	"testing"
	"time"
)

func TestEnqueueKeepsTheNewestWaitingCommit(t *testing.T) {
	var mu sync.Mutex
	var got []string
	started := make(chan struct{})
	release := make(chan struct{})
	svc := &Service{
		cfg: Config{Remote: "https://example.com/site.git"},
		execute: func(ctx context.Context, remote, sha string) {
			mu.Lock()
			got = append(got, sha)
			first := len(got) == 1
			mu.Unlock()
			if first {
				close(started)
				<-release
			}
		},
	}
	svc.Enqueue("", "one")
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("deploy did not start")
	}
	svc.Enqueue("", "two")
	svc.Enqueue("", "three")
	svc.Enqueue("", "one")
	close(release)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		svc.mu.Lock()
		running := svc.running
		svc.mu.Unlock()
		if !running {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	svc.mu.Lock()
	running := svc.running
	svc.mu.Unlock()
	if running {
		t.Fatal("deploy still running")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 || got[0] != "one" || got[1] != "three" {
		t.Fatalf("ran %v", got)
	}
}

func TestDecideHook(t *testing.T) {
	svc := testService(t, "https://example.com/site.git")
	body := []byte(`{"ref":"refs/heads/main","after":"0123456789abcdef0123456789abcdef01234567"}`)
	status, remote, sha := svc.DecideHook("push", "sha256=dead", body)
	if status != http.StatusUnauthorized || sha != "" || remote != "" {
		t.Fatalf("bad signature: %d %s", status, sha)
	}
	status, remote, sha = svc.DecideHook("push", sign("secret", body), body)
	if status != http.StatusAccepted || sha != "0123456789abcdef0123456789abcdef01234567" || remote != "" {
		t.Fatalf("push: %d %s %s", status, remote, sha)
	}
	upper := []byte(`{"ref":"refs/heads/main","after":"0123456789ABCDEF0123456789ABCDEF01234567"}`)
	status, remote, sha = svc.DecideHook("push", sign("secret", upper), upper)
	if status != http.StatusAccepted || sha != "0123456789abcdef0123456789abcdef01234567" || remote != "" {
		t.Fatalf("upper push: %d %s", status, sha)
	}
	other := []byte(`{"ref":"refs/heads/other","after":"0123456789abcdef0123456789abcdef01234567"}`)
	status, _, sha = svc.DecideHook("push", sign("secret", other), other)
	if status != http.StatusNoContent || sha != "" {
		t.Fatalf("other branch: %d %s", status, sha)
	}
	status, _, _ = svc.DecideHook("ping", sign("secret", []byte(`{}`)), []byte(`{}`))
	if status != http.StatusNoContent {
		t.Fatalf("ping: %d", status)
	}
	if _, err := ParseDeploySHA([]byte(`{"sha":"nope"}`)); err == nil {
		t.Fatal("expected invalid sha")
	}
	got, err := ParseDeploySHA([]byte(`{"sha":"0123456789abcdef0123456789abcdef01234567"}`))
	if err != nil || got != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("sha: %s %v", got, err)
	}
}

func TestPublishChangedApp(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	origin := t.TempDir()
	gitCmd(t, origin, "init", "-b", "main")
	writeApp(t, origin, "hello", "one")
	gitCmd(t, origin, "add", ".")
	gitCmd(t, origin, "commit", "-m", "first")
	first := strings.TrimSpace(gitCmd(t, origin, "rev-parse", "HEAD"))

	writeApp(t, origin, "hello", "two")
	gitCmd(t, origin, "add", ".")
	gitCmd(t, origin, "commit", "-m", "second")
	second := strings.TrimSpace(gitCmd(t, origin, "rev-parse", "HEAD"))

	var built []string
	workspace := t.TempDir()
	builder := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer builder" {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		var req struct {
			Apps []BuildApp `json:"apps"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad", http.StatusBadRequest)
			return
		}
		for _, app := range req.Apps {
			built = append(built, app.Slug)
			writeFile(t, filepath.Join(workspace, "out", "apps", app.Slug, "dist", "index.html"), "built-"+app.Slug)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "log": "built\n"})
	}))
	defer builder.Close()

	siteRoot := t.TempDir()
	writeFile(t, filepath.Join(siteRoot, "apps.yaml"), registry("hello"))
	writeFile(t, filepath.Join(siteRoot, "apps", "hello", "dist", "index.html"), "old")
	store := &memStore{}
	store.seed("", first)
	svc := testService(t, origin)
	svc.cfg.BuilderURL = builder.URL
	svc.cfg.Workspace = workspace
	svc.cfg.SiteRoot = siteRoot
	svc.repo.Dir = filepath.Join(workspace, "src")
	svc.repo.Remote = origin
	svc.store = store

	svc.perform(context.Background(), "", second)
	body, err := os.ReadFile(filepath.Join(siteRoot, "apps", "hello", "dist", "index.html"))
	if err != nil || string(body) != "built-hello" {
		t.Fatalf("dist: %q %v", body, err)
	}
	if len(built) != 1 || built[0] != "hello" {
		t.Fatalf("built %v", built)
	}
	latest, err := store.Latest(context.Background(), "")
	if err != nil || latest == nil || latest.Status != "published" || latest.SHA != second {
		t.Fatalf("latest: %+v %v", latest, err)
	}

	svc.perform(context.Background(), "", second)
	again, err := store.Latest(context.Background(), "")
	if err != nil || again == nil || again.Status != "published" || !strings.Contains(again.Log, "already published") {
		t.Fatalf("second run: %+v %v", again, err)
	}
	if len(store.runs) != 3 {
		t.Fatalf("runs: %d", len(store.runs))
	}
}

func TestFailedBuildLeavesTheLiveSite(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	origin := t.TempDir()
	gitCmd(t, origin, "init", "-b", "main")
	writeApp(t, origin, "hello", "one")
	gitCmd(t, origin, "add", ".")
	gitCmd(t, origin, "commit", "-m", "first")
	sha := strings.TrimSpace(gitCmd(t, origin, "rev-parse", "HEAD"))

	builder := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "log": "npm failed\n"})
	}))
	defer builder.Close()

	workspace := t.TempDir()
	siteRoot := t.TempDir()
	writeFile(t, filepath.Join(siteRoot, "apps.yaml"), registry("hello"))
	writeFile(t, filepath.Join(siteRoot, "apps", "hello", "dist", "index.html"), "old")
	svc := testService(t, origin)
	svc.cfg.BuilderURL = builder.URL
	svc.cfg.Workspace = workspace
	svc.repo.Dir = filepath.Join(workspace, "src")
	svc.repo.Remote = origin
	svc.cfg.SiteRoot = siteRoot
	svc.perform(context.Background(), "", "")
	body, err := os.ReadFile(filepath.Join(siteRoot, "apps", "hello", "dist", "index.html"))
	if err != nil || string(body) != "old" {
		t.Fatalf("dist: %q %v", body, err)
	}
	latest, err := svc.store.Latest(context.Background(), "")
	if err != nil || latest == nil || latest.Status != "failed" || latest.SHA != sha {
		t.Fatalf("latest: %+v %v", latest, err)
	}
}

func testService(t *testing.T, remote string) *Service {
	t.Helper()
	svc, err := New(Config{
		Remote:        remote,
		Branch:        "main",
		WebhookSecret: "secret",
		DeployToken:   "token",
		BuilderToken:  "builder",
		BuilderURL:    "http://127.0.0.1:1",
		Workspace:     t.TempDir(),
		SiteRoot:      t.TempDir(),
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

func writeApp(t *testing.T, root, slug, source string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "apps.yaml"), registry(slug))
	writeFile(t, filepath.Join(root, "apps", slug, "package.json"), "{}")
	writeFile(t, filepath.Join(root, "apps", slug, "index.js"), source)
}

func gitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	base := []string{"-c", "commit.gpgsign=false", "-c", "user.email=test@example.com", "-c", "user.name=test"}
	cmd := exec.Command("git", append(base, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}
