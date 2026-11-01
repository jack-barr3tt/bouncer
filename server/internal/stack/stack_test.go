package stack

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMergeEnvKeepsExistingValues(t *testing.T) {
	existing := "SITE_PATH=/var/www/apps\nPORT=9090\n"
	template := "SITE_PATH=/var/www/example\nPORT=8080\nRUSTFS_ACCESS_KEY=BOUNCERLOCAL\n"
	got, added := MergeEnv(existing, template)
	if strings.Contains(got, "SITE_PATH=/var/www/example") || !strings.Contains(got, "SITE_PATH=/var/www/apps") {
		t.Fatalf("site path changed: %q", got)
	}
	if strings.Contains(got, "PORT=8080") || !strings.Contains(got, "PORT=9090") {
		t.Fatalf("port changed: %q", got)
	}
	if !strings.Contains(got, "RUSTFS_ACCESS_KEY=BOUNCERLOCAL\n") {
		t.Fatalf("missing storage key: %q", got)
	}
	if len(added) != 1 || added[0] != "RUSTFS_ACCESS_KEY" {
		t.Fatalf("added: %v", added)
	}
	if !strings.HasPrefix(got, existing) {
		t.Fatalf("existing order changed: %q", got)
	}
}

func TestAdoptReplacesLegacyCompose(t *testing.T) {
	site := t.TempDir()
	share := t.TempDir()
	deploy := filepath.Join(site, "deploy")
	if err := os.MkdirAll(deploy, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := "services:\n  postgres:\n    image: postgres:17\n"
	if err := os.WriteFile(filepath.Join(deploy, "docker-compose.yml"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deploy, ".env"), []byte("SITE_PATH=/var/www/apps\n# kept\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stub := "# bouncer-reconcile\nservices:\n  reconcile:\n    image: bouncer\n"
	if err := os.WriteFile(filepath.Join(share, "docker-compose.yml"), []byte(stub), 0o644); err != nil {
		t.Fatal(err)
	}
	example := "- name: SITE_PATH\n  value: \"/tmp/site\"\n- name: RUSTFS_ACCESS_KEY\n  value: \"BOUNCERLOCAL\"\n"
	if err := os.WriteFile(filepath.Join(share, "env.yaml"), []byte(example), 0o644); err != nil {
		t.Fatal(err)
	}

	msg, err := Adopt(Options{SiteRoot: site, ShareDir: share})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(msg, "docker compose up -d") {
		t.Fatalf("message: %q", msg)
	}
	got, err := os.ReadFile(filepath.Join(deploy, "docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != stub {
		t.Fatalf("compose: %q", got)
	}
	env, err := os.ReadFile(filepath.Join(deploy, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(env)
	if !strings.Contains(text, "SITE_PATH=/var/www/apps") || !strings.Contains(text, "RUSTFS_ACCESS_KEY=BOUNCERLOCAL") {
		t.Fatalf("env: %q", text)
	}
	if strings.Contains(text, "SITE_PATH=/tmp/site") {
		t.Fatalf("template overwrote site path: %q", text)
	}

	again, err := Adopt(Options{SiteRoot: site, ShareDir: share})
	if err != nil {
		t.Fatal(err)
	}
	if again != "" {
		t.Fatalf("second adopt: %q", again)
	}
}

func TestReconcileAppliesStackThenStops(t *testing.T) {
	deploy := t.TempDir()
	share := t.TempDir()
	if err := os.WriteFile(filepath.Join(deploy, ".env"), []byte("SITE_PATH=/var/www/apps\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(share, "env.yaml"), []byte("- name: RUSTFS_ACCESS_KEY\n  value: \"BOUNCERLOCAL\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stackFile := filepath.Join(share, "stack.yml")
	if err := os.WriteFile(stackFile, []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var mu sync.Mutex
	var calls [][]string
	started := make(chan struct{})
	run := func(ctx context.Context, name string, args ...string) error {
		mu.Lock()
		calls = append(calls, append([]string{name}, args...))
		n := len(calls)
		mu.Unlock()
		if n == 2 {
			close(started)
		}
		return nil
	}

	errc := make(chan error, 1)
	go func() {
		errc <- Reconcile(ctx, Options{DeployDir: deploy, ShareDir: share, Catalog: nopCatalog{}}, run)
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("compose was not applied")
	}
	cancel()
	if err := <-errc; err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 3 {
		t.Fatalf("calls: %v", calls)
	}
	envFile := filepath.Join(deploy, ".env")
	want := [][]string{
		{"docker", "compose", "--project-directory", deploy, "--env-file", envFile, "-f", stackFile, "pull"},
		{"docker", "compose", "--project-directory", deploy, "--env-file", envFile, "-f", stackFile, "up", "-d"},
		{"docker", "compose", "--project-directory", deploy, "--env-file", envFile, "-f", stackFile, "stop"},
	}
	for i, call := range want {
		if strings.Join(calls[i], " ") != strings.Join(call, " ") {
			t.Fatalf("call %d: got %q want %q", i, calls[i], call)
		}
	}
	env, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(env), "SITE_PATH=/var/www/apps") || !strings.Contains(string(env), "RUSTFS_ACCESS_KEY=BOUNCERLOCAL") {
		t.Fatalf("env: %q", env)
	}
}

type nopCatalog struct{}

func (nopCatalog) Latest(context.Context) (string, error) { return "", nil }
func (nopCatalog) Example(context.Context, string) (string, error) {
	return "", nil
}

func TestRenderKeepsARequiredValue(t *testing.T) {
	got := Render([]Field{{Name: "TOKEN", Value: "secret", Required: true}})
	if got != "TOKEN=secret\n" {
		t.Fatalf("rendered: %q", got)
	}
	blank := Render([]Field{{Name: "TOKEN", Required: true}})
	if blank != "TOKEN=\n" {
		t.Fatalf("blank: %q", blank)
	}
}

func TestDeployEnvSpecMarksStorageSecretsRequired(t *testing.T) {
	body, err := os.ReadFile("../../../deploy/env.yaml")
	if err != nil {
		t.Fatal(err)
	}
	fields, err := ParseSpec(string(body))
	if err != nil {
		t.Fatal(err)
	}
	required := map[string]bool{}
	for _, field := range fields {
		if field.Required {
			required[field.Name] = true
		}
	}
	want := []string{
		"PORT",
		"SITE_PATH",
		"PUBLIC_BASE_URL",
		"AUTH_BOOTSTRAP_USERNAME",
		"AUTH_BOOTSTRAP_PASSWORD",
		"DEPLOY_WEBHOOK_SECRET",
		"DEPLOY_TOKEN",
		"BUILDER_TOKEN",
		"RUSTFS_ACCESS_KEY",
		"RUSTFS_SECRET_KEY",
	}
	for _, name := range want {
		if !required[name] {
			t.Fatalf("required: %v", required)
		}
	}
	rendered := Render(fields)
	if !strings.Contains(rendered, "RUSTFS_ACCESS_KEY=\n") || strings.Contains(rendered, "RUSTFS_ACCESS_KEY=BOUNCER") {
		t.Fatalf("rendered: %q", rendered)
	}
}

func fields(t *testing.T, text string) []Field {
	t.Helper()
	got, err := ParseSpec(text)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestChooseBlocksAVersionThatNeedsConfig(t *testing.T) {
	env := "IMAGE_TAG=1.0.0\nSITE_PATH=/var/www/apps\n"
	embedded := fields(t, "- name: STORAGE_BUCKET\n  value: \"bouncer\"\n")
	candidate := fields(t, "- name: RUSTFS_ACCESS_KEY\n  required: true\n- name: STORAGE_REGION\n  value: \"us-east-1\"\n")
	got := choose("1.0.0", "1.0.0", env, embedded, "1.1.0", candidate)
	if !got.apply || got.apply && strings.Contains(got.env, "IMAGE_TAG=1.1.0") {
		t.Fatalf("moved the pin: %+v", got)
	}
	if !strings.Contains(got.env, "IMAGE_TAG=1.0.0") || !strings.Contains(got.env, "SITE_PATH=/var/www/apps") {
		t.Fatalf("env: %q", got.env)
	}
	if got.noticeVersion != "1.1.0" || len(got.noticeKeys) != 1 || got.noticeKeys[0] != "RUSTFS_ACCESS_KEY" {
		t.Fatalf("notice: %+v", got)
	}
	if got.replace {
		t.Fatal("reconciler should stay on the current pin")
	}
}

func TestChoosePinsTheNewestVersionWhenConfigIsSet(t *testing.T) {
	env := "IMAGE_TAG=latest\nSITE_PATH=/srv/apps\nRUSTFS_ACCESS_KEY=real-key\n"
	example := fields(t, "- name: RUSTFS_ACCESS_KEY\n  required: true\n- name: STORAGE_BUCKET\n  value: \"bouncer\"\n")
	got := choose("latest", "1.2.0", env, example, "1.3.0", example)
	if !got.apply || !strings.Contains(got.env, "IMAGE_TAG=1.3.0") {
		t.Fatalf("pin: %q", got.env)
	}
	if !strings.Contains(got.env, "SITE_PATH=/srv/apps") || !strings.Contains(got.env, "RUSTFS_ACCESS_KEY=real-key") {
		t.Fatalf("values changed: %q", got.env)
	}
	if !strings.Contains(got.env, "STORAGE_BUCKET=bouncer\n") {
		t.Fatalf("optional key missing: %q", got.env)
	}
	if got.noticeVersion != "" || !got.replace {
		t.Fatalf("choice: %+v", got)
	}
}

func TestChooseHoldsTheRunningVersionWhenItNeedsConfig(t *testing.T) {
	env := "IMAGE_TAG=latest\n"
	example := fields(t, "- name: RUSTFS_SECRET_KEY\n  required: true\n")
	got := choose("latest", "1.4.0", env, example, "", nil)
	if got.apply {
		t.Fatal("stack was applied")
	}
	if !strings.Contains(got.env, "IMAGE_TAG=1.4.0\n") {
		t.Fatalf("pin: %q", got.env)
	}
	if got.noticeVersion != "1.4.0" || len(got.noticeKeys) != 1 || got.noticeKeys[0] != "RUSTFS_SECRET_KEY" {
		t.Fatalf("notice: %+v", got)
	}
}

func TestAutoUpdateIsOnUnlessTurnedOff(t *testing.T) {
	if !AutoUpdate("") || !AutoUpdate("true") || AutoUpdate("false") || AutoUpdate("OFF") {
		t.Fatal("auto update parse")
	}
}

func TestMaintainLeavesThePinWhenUpdatesAreOff(t *testing.T) {
	deploy := t.TempDir()
	share := t.TempDir()
	env := "IMAGE_TAG=1.0.0\nAUTO_UPDATE=false\nRUSTFS_ACCESS_KEY=real-key\n"
	if err := os.WriteFile(filepath.Join(deploy, ".env"), []byte(env), 0o644); err != nil {
		t.Fatal(err)
	}
	spec := "- name: RUSTFS_ACCESS_KEY\n  required: true\n"
	if err := os.WriteFile(filepath.Join(share, "env.yaml"), []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deploy, "update.json"), []byte("{\"version\":\"1.2.0\",\"keys\":[\"RUSTFS_ACCESS_KEY\"]}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := maintain(context.Background(), Options{
		DeployDir: deploy,
		ShareDir:  share,
		Version:   "1.0.0",
		Catalog:   fixedCatalog{version: "1.2.0", example: spec},
	}, func(context.Context, string, ...string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(deploy, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "IMAGE_TAG=1.0.0\n") || strings.Contains(string(got), "IMAGE_TAG=1.2.0") {
		t.Fatalf("pin moved: %q", got)
	}
	if _, err := os.Stat(filepath.Join(deploy, "update.json")); !os.IsNotExist(err) {
		t.Fatalf("notice stayed: %v", err)
	}
}

type fixedCatalog struct {
	version string
	example string
}

func (c fixedCatalog) Latest(context.Context) (string, error) { return c.version, nil }
func (c fixedCatalog) Example(context.Context, string) (string, error) {
	return c.example, nil
}

func TestMaintainSkipsTheStackWhenConfigIsMissing(t *testing.T) {
	deploy := t.TempDir()
	share := t.TempDir()
	if err := os.WriteFile(filepath.Join(deploy, ".env"), []byte("IMAGE_TAG=latest\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(share, "env.yaml"), []byte("- name: RUSTFS_SECRET_KEY\n  required: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	called := false
	err := maintain(context.Background(), Options{DeployDir: deploy, ShareDir: share, Version: "1.4.0", Catalog: nopCatalog{}}, func(context.Context, string, ...string) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("docker was called")
	}
	notice, ok, err := ReadNotice(filepath.Join(deploy, "update.json"))
	if err != nil || !ok {
		t.Fatalf("notice %v %v", notice, err)
	}
	if notice.Version != "1.4.0" || len(notice.Keys) != 1 || notice.Keys[0] != "RUSTFS_SECRET_KEY" {
		t.Fatalf("notice: %+v", notice)
	}
	env, err := os.ReadFile(filepath.Join(deploy, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(env), "IMAGE_TAG=1.4.0\n") {
		t.Fatalf("env: %q", env)
	}
}
