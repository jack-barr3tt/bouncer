package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Exec func(ctx context.Context, dir, name string, args ...string) (string, error)

type Builder struct {
	Workspace string
	Token     string
	Run       Exec
}

func (b *Builder) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /build", b.handleBuild)
	return mux
}

func ListenAndServeBuilder() error {
	token := os.Getenv("BUILDER_TOKEN")
	workspace := strings.TrimSpace(os.Getenv("WORKSPACE"))
	if workspace == "" {
		workspace = "/workspace"
	}
	listen := strings.TrimSpace(os.Getenv("BUILDER_LISTEN"))
	if listen == "" {
		listen = ":8081"
	}
	builder := &Builder{Workspace: workspace, Token: token, Run: defaultExec}
	server := &http.Server{
		Addr:              listen,
		Handler:           builder.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	if token == "" {
		fmt.Fprintln(os.Stderr, "BUILDER_TOKEN is empty; refusing builds")
	}
	return server.ListenAndServe()
}

func (b *Builder) handleBuild(w http.ResponseWriter, r *http.Request) {
	if b.Token == "" || !bearerOK(r.Header.Get("Authorization"), b.Token) {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	var req struct {
		Slugs []string `json:"slugs"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	logText, err := b.Build(r.Context(), req.Slugs)
	status := http.StatusOK
	if err != nil {
		status = http.StatusInternalServerError
		if logText != "" && !strings.HasSuffix(logText, "\n") {
			logText += "\n"
		}
		logText += err.Error()
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": err == nil, "log": logText})
}

func (b *Builder) Build(ctx context.Context, slugs []string) (string, error) {
	run := b.Run
	if run == nil {
		run = defaultExec
	}
	out := filepath.Join(b.Workspace, "out")
	if err := os.RemoveAll(out); err != nil {
		return "", err
	}
	var buf strings.Builder
	seen := map[string]struct{}{}
	for _, slug := range slugs {
		if !safeSlug(slug) {
			return buf.String(), fmt.Errorf("invalid slug %q", slug)
		}
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}
		dir := filepath.Join(b.Workspace, "src", "apps", slug)
		if _, err := os.Stat(filepath.Join(dir, "package.json")); err != nil {
			return buf.String(), fmt.Errorf("%s has no package.json", slug)
		}
		appCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		text, err := run(appCtx, dir, "npm", "ci")
		buf.WriteString(text)
		if err == nil {
			text, err = run(appCtx, dir, "npm", "run", "build")
			buf.WriteString(text)
		}
		cancel()
		if err != nil {
			return buf.String(), err
		}
		dist := filepath.Join(dir, "dist")
		info, statErr := os.Stat(dist)
		if statErr != nil || !info.IsDir() {
			return buf.String(), fmt.Errorf("%s did not produce dist", slug)
		}
		dest := filepath.Join(out, "apps", slug, "dist")
		if err := copyTree(dist, dest); err != nil {
			return buf.String(), err
		}
	}
	return buf.String(), nil
}

func defaultExec(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	text := string(out)
	if err != nil {
		if len(text) > 0 && !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		return text, fmt.Errorf("%s: %w", name, err)
	}
	return text, nil
}

func (s *Service) build(ctx context.Context, slugs []string) (string, error) {
	if err := os.RemoveAll(s.outputDir()); err != nil {
		return "", err
	}
	if err := os.MkdirAll(s.outputDir(), 0o755); err != nil {
		return "", err
	}
	timeout := time.Duration(len(slugs)) * 10 * time.Minute
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	payload, err := json.Marshal(map[string]any{"slugs": slugs})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.cfg.BuilderURL, "/")+"/build", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.BuilderToken)
	req.Header.Set("Content-Type", "application/json")
	res, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("builder: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 256<<10))
	if err != nil {
		return "", err
	}
	var result struct {
		OK  bool   `json:"ok"`
		Log string `json:"log"`
	}
	if jsonErr := json.Unmarshal(body, &result); jsonErr != nil && res.StatusCode == http.StatusOK {
		return string(body), jsonErr
	}
	if res.StatusCode != http.StatusOK || !result.OK {
		logText := result.Log
		if logText == "" {
			logText = string(body)
		}
		return logText, fmt.Errorf("builder failed (%d)", res.StatusCode)
	}
	return result.Log, nil
}
