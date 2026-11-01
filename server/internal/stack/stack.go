package stack

import (
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

const (
	Marker       = "bouncer-reconcile"
	defaultShare = "/opt/bouncer"
)

type Options struct {
	DeployDir string
	ShareDir  string
	SiteRoot  string
	Version   string
	Catalog   Catalog
	Interval  time.Duration
}

func (o Options) shareDir() string {
	if o.ShareDir != "" {
		return o.ShareDir
	}
	if dir := os.Getenv("BOUNCER_SHARE"); dir != "" {
		return dir
	}
	return defaultShare
}

func (o Options) deployDir() string {
	if o.DeployDir != "" {
		return o.DeployDir
	}
	if dir := os.Getenv("DEPLOY_DIR"); dir != "" {
		return dir
	}
	return "/deploy"
}

func (o Options) siteRoot() string {
	if o.SiteRoot != "" {
		return o.SiteRoot
	}
	if dir := os.Getenv("SITE_ROOT"); dir != "" {
		return dir
	}
	return "/site"
}

type Runner func(ctx context.Context, name string, args ...string) error

func Exec(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// MergeEnv appends template keys that existing does not already set.
// Existing values, comments, and order stay in place.
func MergeEnv(existing, template string) (string, []string) {
	have := map[string]struct{}{}
	for _, line := range strings.Split(existing, "\n") {
		if key := envKey(line); key != "" {
			have[key] = struct{}{}
		}
	}
	var added []string
	var extra []string
	for _, line := range strings.Split(template, "\n") {
		key := envKey(line)
		if key == "" {
			continue
		}
		if _, ok := have[key]; ok {
			continue
		}
		have[key] = struct{}{}
		added = append(added, key)
		extra = append(extra, strings.TrimRight(line, "\r"))
	}
	if len(extra) == 0 {
		return existing, nil
	}
	body := existing
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	body += strings.Join(extra, "\n")
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return body, added
}

func envKey(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return ""
	}
	key, _, ok := strings.Cut(line, "=")
	if !ok {
		return ""
	}
	key = strings.TrimSpace(key)
	if key == "" || strings.ContainsAny(key, " \t") {
		return ""
	}
	return key
}

// Adopt replaces a legacy site compose file with the reconciler stub and
// appends env keys the current stack expects. A file that already carries
// the marker is left in place. A missing compose file is left missing.
func Adopt(opts Options) (string, error) {
	deploy := filepath.Join(opts.siteRoot(), "deploy")
	composePath := filepath.Join(deploy, "docker-compose.yml")
	body, err := os.ReadFile(composePath)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if strings.Contains(string(body), Marker) {
		return "", nil
	}
	stub, err := os.ReadFile(filepath.Join(opts.shareDir(), "docker-compose.yml"))
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(composePath, stub, 0o644); err != nil {
		return "", err
	}
	example, err := os.ReadFile(filepath.Join(opts.shareDir(), "env.yaml"))
	if err != nil {
		return "", err
	}
	fields, err := ParseSpec(string(example))
	if err != nil {
		return "", err
	}
	rendered := Render(fields)
	if _, err := mergeEnvFile(filepath.Join(deploy, ".env"), rendered, true); err != nil {
		return "", err
	}
	if _, err := mergeEnvFile(filepath.Join(deploy, ".env.example"), rendered, false); err != nil {
		return "", err
	}
	log.Printf("updated %s", composePath)
	return "Updated deploy/docker-compose.yml. From deploy/, run: docker compose up -d", nil
}

// Reconcile applies the pinned stack, then keeps checking for a newer published
// version. A version whose required env keys are missing is left unapplied and
// recorded for the admin page. Otherwise the pin moves to that version.
func Reconcile(ctx context.Context, opts Options, run Runner) error {
	if run == nil {
		run = Exec
	}
	replacing.Store(false)
	if opts.Catalog == nil {
		opts.Catalog = liveCatalog(run)
	}
	if opts.Version == "" {
		opts.Version = os.Getenv("BOUNCER_VERSION")
	}
	interval := time.Minute
	if opts.Interval > 0 {
		interval = opts.Interval
	}
	if err := maintain(ctx, opts, run); err != nil && ctx.Err() == nil {
		log.Printf("update: %v", err)
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			if replacing.Load() {
				return nil
			}
			return compose(context.Background(), run, opts.deployDir(), filepath.Join(opts.shareDir(), "stack.yml"), "stop")
		case <-ticker.C:
			if err := maintain(ctx, opts, run); err != nil {
				log.Printf("update: %v", err)
			}
		}
	}
}

var replacing atomic.Bool

func maintain(ctx context.Context, opts Options, run Runner) error {
	deploy := opts.deployDir()
	share := opts.shareDir()
	envPath := filepath.Join(deploy, ".env")
	envText, err := os.ReadFile(envPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(share, "env.yaml"))
	if err != nil {
		return err
	}
	embedded, err := ParseSpec(string(raw))
	if err != nil {
		return err
	}
	candidate, candidateFields := "", []Field(nil)
	if opts.Catalog != nil {
		latest, err := opts.Catalog.Latest(ctx)
		if err != nil {
			log.Printf("update: %v", err)
		} else if latest != "" {
			example, err := opts.Catalog.Example(ctx, latest)
			if err != nil {
				log.Printf("update: %v", err)
			} else if fields, err := ParseSpec(example); err != nil {
				log.Printf("update: %v", err)
			} else {
				candidate, candidateFields = latest, fields
			}
		}
	}
	values := envValues(string(envText))
	if !AutoUpdate(values["AUTO_UPDATE"]) {
		candidate, candidateFields = "", nil
	}
	decision := choose(values["IMAGE_TAG"], opts.Version, string(envText), embedded, candidate, candidateFields)
	noticePath := filepath.Join(deploy, "update.json")
	if err := writeNotice(noticePath, Notice{Version: decision.noticeVersion, Keys: decision.noticeKeys}); err != nil {
		return err
	}
	if len(decision.noticeKeys) > 0 {
		log.Printf("bouncer %s needs %s in %s", decision.noticeVersion, strings.Join(decision.noticeKeys, ", "), envPath)
	}
	if decision.env != string(envText) {
		if err := os.MkdirAll(deploy, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(envPath, []byte(decision.env), 0o644); err != nil {
			return err
		}
	}
	if !decision.apply {
		return nil
	}
	stackFile := filepath.Join(share, "stack.yml")
	if err := compose(ctx, run, deploy, stackFile, "pull"); err != nil {
		return err
	}
	upErr := compose(ctx, run, deploy, stackFile, "up", "-d")
	if !decision.replace {
		return upErr
	}
	replacing.Store(true)
	replaceErr := compose(context.Background(), run, deploy, filepath.Join(deploy, "docker-compose.yml"), "up", "-d")
	if upErr != nil && replaceErr != nil {
		return errors.Join(upErr, replaceErr)
	}
	if replaceErr != nil {
		return replaceErr
	}
	return upErr
}

func compose(ctx context.Context, run Runner, deployDir, stackFile string, extra ...string) error {
	args := []string{
		"compose",
		"--project-directory", deployDir,
		"--env-file", filepath.Join(deployDir, ".env"),
		"-f", stackFile,
	}
	args = append(args, extra...)
	return run(ctx, "docker", args...)
}

func mergeEnvFile(path, template string, create bool) ([]string, error) {
	existing, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if !create {
			return nil, nil
		}
		existing = nil
		err = nil
	}
	if err != nil {
		return nil, err
	}
	next, added := MergeEnv(string(existing), template)
	if len(added) == 0 {
		return nil, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(next), 0o644); err != nil {
		return nil, err
	}
	return added, nil
}
