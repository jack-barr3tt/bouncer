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
}

type sourceRepo struct {
	Name   string
	Remote string
	Branch string
	Commit string
	Apps   []sourceApp
}

func (s *Service) discover() ([]sourceRepo, error) {
	if strings.TrimSpace(s.cfg.SiteRoot) == "" {
		return nil, nil
	}
	opened, err := site.Open(s.cfg.SiteRoot)
	if err != nil {
		return nil, err
	}
	file, err := opened.Load()
	if err != nil {
		return nil, err
	}
	groups := make(map[string]*sourceRepo, len(file.Repos))
	order := make([]string, 0, len(file.Repos))
	for _, repo := range file.Repos {
		groups[repo.Name] = &sourceRepo{Name: repo.Name, Remote: repo.Remote, Branch: repo.Branch, Commit: repo.Commit}
		order = append(order, repo.Name)
	}
	for _, app := range file.Apps {
		if app.Repo == "" {
			continue
		}
		repo := groups[app.Repo]
		if repo == nil {
			return nil, fmt.Errorf("%s: repo %q is not listed", app.Slug, app.Repo)
		}
		rel := app.Dir
		if rel == "" {
			rel = "."
		}
		repo.Apps = append(repo.Apps, sourceApp{Slug: app.Slug, Rel: rel})
	}
	out := make([]sourceRepo, 0, len(order))
	for _, name := range order {
		out = append(out, *groups[name])
	}
	return out, nil
}

func (s *Service) checkout(repo sourceRepo) Repo {
	return Repo{
		Dir:        filepath.Join(s.cfg.Workspace, "src", "repos", repo.Name),
		Remote:     repo.Remote,
		Branch:     repo.Branch,
		KeyFile:    s.cfg.KeyFile,
		RuntimeDir: s.cfg.RuntimeDir,
	}
}

func (s *Service) performSources(ctx context.Context, remote, requested string) {
	if s.store == nil {
		log.Printf("deploy: database is required when a repository is configured")
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
		fmt.Fprintf(&buf, "BUILDER_TOKEN is required when a repository is configured\n")
		return
	}
	if repo.Commit != "" {
		if requested != "" && !strings.EqualFold(requested, repo.Commit) {
			fmt.Fprintf(&buf, "commit is pinned to %s\n", repo.Commit)
			return
		}
		requested = repo.Commit
	}
	checkout := s.checkout(repo)
	var syncErr error
	resolved, syncErr = checkout.Sync(ctx, requested)
	if syncErr != nil {
		resolved = requested
		fmt.Fprintf(&buf, "%s\n", syncErr)
		log.Printf("deploy %s failed: %v", repo.Name, syncErr)
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
	fmt.Fprintf(&buf, "repo %s\n", repo.Name)
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
				sourced = append(sourced, sourcedApp(app))
			}
			for _, slug := range AppsFromDiff(diff, sourced) {
				need[slug] = struct{}{}
			}
		}
	}
	for _, app := range repo.Apps {
		info, err := os.Stat(filepath.Join(s.cfg.SiteRoot, "apps", app.Slug, "dist"))
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
			Dir:   "apps/" + app.Slug,
		})
	}
	return out, note, nil
}

type plannedApp struct {
	Build BuildApp
	Dir   string
}

func (s *Service) Target(remote string) (string, error) {
	repos, err := s.discover()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(remote) == "" {
		if s.cfg.Remote != "" {
			return "", nil
		}
		if len(repos) == 1 && repos[0].Remote != "" {
			return repos[0].Remote, nil
		}
		if len(repos) == 0 {
			return "", fmt.Errorf("unknown remote")
		}
		return "", fmt.Errorf("remote is required")
	}
	if repo, ok := matchRepo(repos, remote); ok && repo.Remote != "" {
		return repo.Remote, nil
	}
	if s.cfg.Remote != "" && sameRemote(remote, s.cfg.Remote) {
		return "", nil
	}
	return "", fmt.Errorf("unknown remote")
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
