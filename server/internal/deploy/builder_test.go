package deploy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuilderCopiesDist(t *testing.T) {
	root := t.TempDir()
	app := filepath.Join(root, "src", "apps", "hello")
	writeFile(t, filepath.Join(app, "package.json"), "{}")
	builder := &Builder{
		Workspace: root,
		Token:     "builder",
		Run: func(ctx context.Context, dir, name string, args ...string) (string, error) {
			if name == "npm" && len(args) > 0 && args[0] == "run" {
				writeFile(t, filepath.Join(dir, "dist", "index.html"), "hi")
			}
			return name + "\n", nil
		},
	}
	logText, err := builder.Build(context.Background(), []BuildApp{{Slug: "hello", Dir: "src/apps/hello"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logText, "npm") {
		t.Fatalf("log: %s", logText)
	}
	body, err := os.ReadFile(filepath.Join(root, "out", "apps", "hello", "dist", "index.html"))
	if err != nil || string(body) != "hi" {
		t.Fatalf("dist: %q %v", body, err)
	}
	if _, err := builder.Build(context.Background(), []BuildApp{{Slug: "../hello", Dir: "src/apps/hello"}}); err == nil {
		t.Fatal("expected invalid slug")
	}
	if _, err := builder.Build(context.Background(), []BuildApp{{Slug: "hello", Dir: "../hello"}}); err == nil {
		t.Fatal("expected invalid dir")
	}
}

func TestBuilderRequiresToken(t *testing.T) {
	builder := &Builder{Workspace: t.TempDir(), Token: "builder", Run: func(ctx context.Context, dir, name string, args ...string) (string, error) {
		return "", nil
	}}
	req := httptest.NewRequest(http.MethodPost, "/build", strings.NewReader(`{"slugs":[]}`))
	res := httptest.NewRecorder()
	builder.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status: %d", res.Code)
	}
	req = httptest.NewRequest(http.MethodPost, "/build", strings.NewReader(`{"slugs":[]}`))
	req.Header.Set("Authorization", "Bearer builder")
	res = httptest.NewRecorder()
	builder.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status: %d %s", res.Code, res.Body.String())
	}
	var payload struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil || !payload.OK {
		t.Fatalf("body: %s %v", res.Body.String(), err)
	}
}
