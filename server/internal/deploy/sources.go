package deploy

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/jack-barr3tt/bouncer/internal/site"
)

type sourceApp struct {
	Slug string
	Rel  string
	Dir  string
}

type sourceRepo struct {
	Path   string
	Remote string
	Branch string
	Apps   []sourceApp
	Err    string
}

func (s *Service) discover() ([]sourceRepo, error) {
	if strings.TrimSpace(s.cfg.SiteRoot) == "" {
		return nil, nil
	}
	opened, err := site.Open(s.cfg.SiteRoot, "")
	if err != nil {
		return nil, err
	}
	apps, err := opened.Apps()
	if err != nil {
		return nil, err
	}
	groups := map[string]*sourceRepo{}
	var order []string
	for _, app := range apps {
		if app.Source == "" {
			continue
		}
		root, rel, found, err := gitRoot(s.cfg.SiteRoot, app.Source)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", app.Slug, err)
		}
		if !found {
			continue
		}
		path, err := filepath.Rel(s.cfg.SiteRoot, root)
		if err != nil || path == ".." || strings.HasPrefix(path, ".."+string(os.PathSeparator)) {
			return nil, fmt.Errorf("%s: source escapes the site", app.Slug)
		}
		key := filepath.ToSlash(path)
		if key == "." {
			key = ""
		}
		repo := groups[key]
		if repo == nil {
			remote, branch, readErr := readClone(context.Background(), root, s.cfg.RuntimeDir, s.cfg.KeyFile)
			repo = &sourceRepo{Path: displayPath(key), Remote: remote, Branch: branch}
			if readErr != nil {
				repo.Err = readErr.Error()
			}
			groups[key] = repo
			order = append(order, key)
		}
		repo.Apps = append(repo.Apps, sourceApp{Slug: app.Slug, Rel: rel, Dir: app.Source})
	}
	out := make([]sourceRepo, 0, len(order))
	for _, key := range order {
		out = append(out, *groups[key])
	}
	return out, nil
}

func displayPath(key string) string {
	if key == "" {
		return "."
	}
	return key
}

func (s *Service) checkout(repo sourceRepo) Repo {
	dir := filepath.Join(s.cfg.Workspace, "src")
	if repo.Path != "" && repo.Path != "." {
		dir = filepath.Join(dir, filepath.FromSlash(repo.Path))
	}
	return Repo{
		Dir:        dir,
		Remote:     repo.Remote,
		Branch:     repo.Branch,
		KeyFile:    s.cfg.KeyFile,
		RuntimeDir: s.cfg.RuntimeDir,
	}
}

func (s *Service) performSources(ctx context.Context, remote, requested string) {
	if s.store == nil {
		log.Printf("deploy: database is required when an app source is set")
		return
	}
	repos, err := s.discover()
	if err != nil {
		s.failSource(ctx, remote, requested, err.Error())
		return
	}
	repo, ok := matchRepo(repos, remote)
	if !ok {
		log.Printf("deploy: unknown remote %s", remote)
		return
	}
	remote = repo.Remote
	id, err := s.store.Start(ctx, remote, requested)
	if err != nil {
		log.Printf("deploy %s: %v", remote, err)
		return
	}
	var buf strings.Builder
	status := "failed"
	resolved := requested
	defer func() {
		if err := s.store.Finish(ctx, id, resolved, status, capLog(buf.String())); err != nil {
			log.Printf("deploy finish: %v", err)
		}
	}()
	if strings.TrimSpace(s.cfg.BuilderToken) == "" {
		fmt.Fprintf(&buf, "BUILDER_TOKEN is required when an app source is set\n")
		return
	}
	if repo.Err != "" {
		fmt.Fprintf(&buf, "%s\n", repo.Err)
		return
	}
	checkout := s.checkout(repo)
	var syncErr error
	resolved, syncErr = checkout.Sync(ctx, requested)
	if syncErr != nil {
		resolved = requested
		fmt.Fprintf(&buf, "%s\n", syncErr)
		log.Printf("deploy %s failed: %v", repo.Path, syncErr)
		return
	}
	last, err := s.store.LastPublished(ctx, remote)
	if err != nil {
		fmt.Fprintf(&buf, "%s\n", err)
		return
	}
	apps, note, err := s.changedSourceApps(ctx, checkout, repo, last, resolved)
	if note != "" {
		fmt.Fprintf(&buf, "%s\n", note)
	}
	if err != nil {
		fmt.Fprintf(&buf, "%s\n", err)
		return
	}
	if len(apps) == 0 {
		fmt.Fprintf(&buf, "already published\n")
		status = "published"
		return
	}
	fmt.Fprintf(&buf, "deploy %s\n", resolved)
	fmt.Fprintf(&buf, "repo %s\n", repo.Path)
	names := make([]string, 0, len(apps))
	builds := make([]BuildApp, 0, len(apps))
	targets := make([]DistTarget, 0, len(apps))
	for _, app := range apps {
		names = append(names, app.Build.Slug)
		builds = append(builds, app.Build)
		targets = append(targets, DistTarget{Slug: app.Build.Slug, Dir: app.Dir})
	}
	fmt.Fprintf(&buf, "apps: %s\n", strings.Join(names, ", "))
	buildLog, err := s.build(ctx, builds)
	buf.WriteString(buildLog)
	if buildLog != "" && !strings.HasSuffix(buildLog, "\n") {
		buf.WriteByte('\n')
	}
	if err != nil {
		fmt.Fprintf(&buf, "%s\n", err)
		return
	}
	if err := PublishDists(s.cfg.SiteRoot, s.outputDir(), targets); err != nil {
		fmt.Fprintf(&buf, "%s\n", err)
		return
	}
	status = "published"
	fmt.Fprintf(&buf, "published\n")
}

func (s *Service) failSource(ctx context.Context, remote, requested, message string) {
	id, err := s.store.Start(ctx, remote, requested)
	if err != nil {
		log.Printf("deploy: %s", message)
		return
	}
	if err := s.store.Finish(ctx, id, requested, "failed", capLog(message+"\n")); err != nil {
		log.Printf("deploy finish: %v", err)
	}
}

func (s *Service) changedSourceApps(ctx context.Context, checkout Repo, repo sourceRepo, last, sha string) ([]plannedApp, string, error) {
	need := map[string]struct{}{}
	var note string
	if ShouldBuildAll(last) {
		for _, app := range repo.Apps {
			need[app.Slug] = struct{}{}
		}
	} else {
		diff, err := checkout.DiffNames(ctx, last, sha)
		if err != nil {
			note = "diff failed, building every app: " + err.Error()
			for _, app := range repo.Apps {
				need[app.Slug] = struct{}{}
			}
		} else {
			sourced := make([]sourcedApp, 0, len(repo.Apps))
			for _, app := range repo.Apps {
				sourced = append(sourced, sourcedApp{Slug: app.Slug, Rel: app.Rel})
			}
			for _, slug := range AppsFromDiff(diff, sourced) {
				need[slug] = struct{}{}
			}
		}
	}
	for _, app := range repo.Apps {
		info, err := os.Stat(filepath.Join(s.cfg.SiteRoot, filepath.FromSlash(app.Dir), "dist"))
		if err != nil || !info.IsDir() {
			need[app.Slug] = struct{}{}
		}
	}
	var out []plannedApp
	for _, app := range repo.Apps {
		if _, ok := need[app.Slug]; !ok {
			continue
		}
		dir := checkout.Dir
		if app.Rel != "." {
			dir = filepath.Join(dir, filepath.FromSlash(app.Rel))
		}
		rel, err := filepath.Rel(s.cfg.Workspace, dir)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return nil, note, fmt.Errorf("build dir for %s escapes the workspace", app.Slug)
		}
		out = append(out, plannedApp{
			Build: BuildApp{Slug: app.Slug, Dir: filepath.ToSlash(rel)},
			Dir:   app.Dir,
		})
	}
	return out, note, nil
}

type plannedApp struct {
	Build BuildApp
	Dir   string
}

func (s *Service) Target(remote string) (string, error) {
	if s.sourceMode() {
		repos, err := s.discover()
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(remote) == "" {
			if len(repos) == 1 {
				if repos[0].Remote == "" {
					return "", fmt.Errorf("remote is required")
				}
				return repos[0].Remote, nil
			}
			return "", fmt.Errorf("remote is required")
		}
		repo, ok := matchRepo(repos, remote)
		if !ok || repo.Remote == "" {
			return "", fmt.Errorf("unknown remote")
		}
		return repo.Remote, nil
	}
	if strings.TrimSpace(remote) != "" && !sameRemote(remote, s.cfg.Remote) {
		return "", fmt.Errorf("unknown remote")
	}
	return "", nil
}

func matchRepo(repos []sourceRepo, remote string) (sourceRepo, bool) {
	if strings.TrimSpace(remote) == "" && len(repos) == 1 {
		return repos[0], true
	}
	for _, repo := range repos {
		if repo.Remote != "" && sameRemote(repo.Remote, remote) {
			return repo, true
		}
	}
	return sourceRepo{}, false
}

func gitRoot(siteRoot, source string) (string, string, bool, error) {
	abs := filepath.Join(siteRoot, filepath.FromSlash(source))
	if !within(siteRoot, abs) {
		return "", "", false, fmt.Errorf("source %q escapes the site", source)
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", "", false, fmt.Errorf("source %q is not a directory", source)
	}
	appsDir := filepath.Join(siteRoot, "apps")
	if !within(appsDir, abs) || abs == appsDir {
		return "", "", false, fmt.Errorf("source %q must be a directory inside apps/", source)
	}
	dir := abs
	for {
		if !within(appsDir, dir) || dir == appsDir {
			return "", "", false, nil
		}
		gitPath := filepath.Join(dir, ".git")
		if st, statErr := os.Stat(gitPath); statErr == nil && (st.IsDir() || st.Mode().IsRegular()) {
			rel, relErr := filepath.Rel(dir, abs)
			if relErr != nil || !within(dir, abs) {
				return "", "", false, fmt.Errorf("source %q escapes the clone", source)
			}
			if rel == "." {
				return dir, ".", true, nil
			}
			return dir, filepath.ToSlash(rel), true, nil
		}
		dir = filepath.Dir(dir)
	}
}

func within(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func readClone(ctx context.Context, clone, runtime, key string) (string, string, error) {
	probe := Repo{Dir: clone, RuntimeDir: runtime, KeyFile: key}
	out, err := probe.git(ctx, clone, "remote", "get-url", "origin")
	if err != nil {
		return "", "", fmt.Errorf("origin: %w", err)
	}
	remote := strings.TrimSpace(out)
	if !validRemote(remote) {
		return "", "", fmt.Errorf("origin is not valid")
	}
	branchOut, err := probe.git(ctx, clone, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return remote, "", err
	}
	branch := strings.TrimSpace(branchOut)
	if branch == "" || branch == "HEAD" {
		return remote, "", fmt.Errorf("detached HEAD")
	}
	if !validBranch(branch) {
		return remote, "", fmt.Errorf("branch is not valid")
	}
	return remote, branch, nil
}

func sameRemote(a, b string) bool {
	left := normalizeRemote(a)
	right := normalizeRemote(b)
	return left != "" && left == right
}

func normalizeRemote(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, "/")
	raw = strings.TrimSuffix(raw, ".git")
	if strings.HasPrefix(raw, "git@") {
		rest := strings.TrimPrefix(raw, "git@")
		host, repoPath, ok := strings.Cut(rest, ":")
		if !ok {
			return strings.ToLower(raw)
		}
		return strings.ToLower(host) + "/" + strings.Trim(repoPath, "/")
	}
	parsed, err := url.Parse(raw)
	if err == nil && parsed.Host != "" {
		repoPath := strings.Trim(strings.TrimSuffix(parsed.Path, ".git"), "/")
		return strings.ToLower(parsed.Hostname()) + "/" + repoPath
	}
	return raw
}
