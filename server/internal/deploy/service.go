package deploy

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const maxLog = 64 * 1024

const defaultKeyFile = "/run/bouncer/git-key"

type Config struct {
	Remote        string
	Branch        string
	KeyFile       string
	WebhookSecret string
	DeployToken   string
	PollInterval  time.Duration
	BuilderURL    string
	BuilderToken  string
	Workspace     string
	SiteRoot      string
	PublicBase    string
	RuntimeDir    string
	Pool          *pgxpool.Pool
	store         history
}

type Status struct {
	Enabled    bool
	Remote     string
	Branch     string
	WebhookURL string
	Latest     *Run
	Repos      []RepoStatus
}

type RepoStatus struct {
	Path   string
	Remote string
	Branch string
	Latest *Run
}

type deployJob struct {
	remote string
	sha    string
}

type Service struct {
	cfg     Config
	store   history
	repo    Repo
	client  *http.Client
	execute func(context.Context, string, string)
	mu      sync.Mutex
	running bool
	active  deployJob
	waiting []deployJob
}

func New(cfg Config) (*Service, error) {
	if cfg.Branch == "" {
		cfg.Branch = "main"
	}
	if cfg.Workspace == "" {
		cfg.Workspace = "/workspace"
	}
	if cfg.BuilderURL == "" {
		cfg.BuilderURL = "http://builder:8081"
	}
	if cfg.KeyFile == "" {
		cfg.KeyFile = defaultKeyFile
	}
	if cfg.RuntimeDir == "" {
		cfg.RuntimeDir = "/run/bouncer"
	}
	cfg.PublicBase = strings.TrimRight(strings.TrimSpace(cfg.PublicBase), "/")
	cfg.Remote = strings.TrimSpace(cfg.Remote)
	if strings.Contains(cfg.RuntimeDir, "'") {
		return nil, fmt.Errorf("runtime dir is not valid")
	}
	if cfg.Remote != "" {
		if !validRemote(cfg.Remote) {
			return nil, fmt.Errorf("GIT_REMOTE is not valid")
		}
		if !validBranch(cfg.Branch) {
			return nil, fmt.Errorf("GIT_BRANCH is not valid")
		}
		if strings.TrimSpace(cfg.BuilderToken) == "" {
			return nil, fmt.Errorf("BUILDER_TOKEN is required when GIT_REMOTE is set")
		}
		if cfg.store == nil && cfg.Pool == nil {
			return nil, fmt.Errorf("database is required when GIT_REMOTE is set")
		}
	}
	if cfg.store == nil && cfg.Pool != nil {
		cfg.store = &pgStore{pool: cfg.Pool}
	}
	svc := &Service{
		cfg:   cfg,
		store: cfg.store,
		client: &http.Client{},
		repo: Repo{
			Dir:        filepath.Join(cfg.Workspace, "src"),
			Remote:     cfg.Remote,
			Branch:     cfg.Branch,
			KeyFile:    cfg.KeyFile,
			RuntimeDir: cfg.RuntimeDir,
		},
	}
	svc.execute = svc.run
	if cfg.Remote == "" && cfg.SiteRoot != "" {
		repos, err := svc.discover()
		if err == nil && len(repos) > 0 {
			if strings.TrimSpace(cfg.BuilderToken) == "" {
				return nil, fmt.Errorf("BUILDER_TOKEN is required when an app source is set")
			}
			if svc.store == nil {
				return nil, fmt.Errorf("database is required when an app source is set")
			}
		}
	}
	return svc, nil
}

func (s *Service) Enabled() bool {
	if s == nil {
		return false
	}
	if s.cfg.Remote != "" {
		return true
	}
	repos, err := s.discover()
	return err == nil && len(repos) > 0
}

func (s *Service) sourceMode() bool {
	repos, err := s.discover()
	return err == nil && len(repos) > 0
}

func (s *Service) WebhookEnabled() bool {
	return s.Enabled() && strings.TrimSpace(s.cfg.WebhookSecret) != ""
}

func (s *Service) TokenEnabled() bool {
	return s.Enabled() && strings.TrimSpace(s.cfg.DeployToken) != ""
}

func (s *Service) Branch() string { return s.cfg.Branch }

func (s *Service) AcceptBearer(header string) bool {
	return bearerOK(header, strings.TrimSpace(s.cfg.DeployToken))
}

func (s *Service) Status(ctx context.Context) (Status, error) {
	out := Status{
		Enabled:    s.Enabled(),
		WebhookURL: s.cfg.PublicBase + "/api/hooks/git",
		Repos:      []RepoStatus{},
	}
	if !out.Enabled {
		out.WebhookURL = ""
		return out, nil
	}
	if s.sourceMode() {
		repos, err := s.discover()
		if err != nil {
			return Status{}, err
		}
		for _, repo := range repos {
			item := RepoStatus{Path: repo.Path, Remote: repo.Remote, Branch: repo.Branch}
			if s.store != nil && repo.Remote != "" {
				latest, err := s.store.Latest(ctx, repo.Remote)
				if err != nil {
					return Status{}, err
				}
				item.Latest = latest
			}
			out.Repos = append(out.Repos, item)
		}
		return out, nil
	}
	out.Remote = s.cfg.Remote
	out.Branch = s.cfg.Branch
	if s.store == nil {
		out.Repos = []RepoStatus{{Remote: s.cfg.Remote, Branch: s.cfg.Branch}}
		return out, nil
	}
	latest, err := s.store.Latest(ctx, "")
	if err != nil {
		return Status{}, err
	}
	out.Latest = latest
	out.Repos = []RepoStatus{{Remote: s.cfg.Remote, Branch: s.cfg.Branch, Latest: latest}}
	return out, nil
}

func (s *Service) Recover(ctx context.Context) error {
	if !s.Enabled() || s.store == nil {
		return nil
	}
	return s.store.FailRunning(ctx)
}

func (s *Service) Poll(ctx context.Context) {
	if !s.Enabled() || s.cfg.PollInterval <= 0 {
		return
	}
	ticker := time.NewTicker(s.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.pollOnce(ctx); err != nil {
				log.Printf("deploy poll: %v", err)
			}
		}
	}
}

func (s *Service) pollOnce(ctx context.Context) error {
	if s.sourceMode() {
		return s.pollSources(ctx)
	}
	if s.cfg.Remote == "" {
		return nil
	}
	tip, err := s.repo.Tip(ctx)
	if err != nil {
		return err
	}
	last, err := s.store.LastPublished(ctx, "")
	if err != nil {
		return err
	}
	if tip != last {
		s.Enqueue("", tip)
	}
	return nil
}

func (s *Service) pollSources(ctx context.Context) error {
	repos, err := s.discover()
	if err != nil {
		return err
	}
	if s.store == nil {
		return fmt.Errorf("database is required when an app source is set")
	}
	var failed error
	for _, repo := range repos {
		if repo.Err != "" || repo.Remote == "" {
			continue
		}
		checkout := s.checkout(repo)
		tip, err := checkout.Tip(ctx)
		if err != nil {
			failed = err
			log.Printf("deploy poll %s: %v", repo.Path, err)
			continue
		}
		last, err := s.store.LastPublished(ctx, repo.Remote)
		if err != nil {
			return err
		}
		if tip != last {
			s.Enqueue(repo.Remote, tip)
		}
	}
	return failed
}

func (s *Service) Enqueue(remote, sha string) {
	if !s.Enabled() {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		if s.active.remote == remote && s.active.sha == sha {
			return
		}
		s.setWaiting(remote, sha)
		return
	}
	s.running = true
	s.active = deployJob{remote: remote, sha: sha}
	go s.loop(remote, sha)
}

func (s *Service) setWaiting(remote, sha string) {
	for i := range s.waiting {
		if s.waiting[i].remote == remote {
			s.waiting[i].sha = sha
			return
		}
	}
	s.waiting = append(s.waiting, deployJob{remote: remote, sha: sha})
}

func (s *Service) loop(remote, sha string) {
	for {
		s.execute(context.Background(), remote, sha)
		s.mu.Lock()
		if len(s.waiting) == 0 {
			s.running = false
			s.active = deployJob{}
			s.mu.Unlock()
			return
		}
		job := s.waiting[0]
		s.waiting = s.waiting[1:]
		s.active = job
		remote, sha = job.remote, job.sha
		s.mu.Unlock()
	}
}

func (s *Service) run(ctx context.Context, remote, sha string) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Hour)
	defer cancel()
	s.perform(ctx, remote, sha)
}

func (s *Service) perform(ctx context.Context, remote, requested string) {
	if s.sourceMode() {
		s.performSources(ctx, remote, requested)
		return
	}
	s.performLegacy(ctx, requested)
}

func (s *Service) performLegacy(ctx context.Context, requested string) {
	id, err := s.store.Start(ctx, "", requested)
	if err != nil {
		log.Printf("deploy %s: %v", requested, err)
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
	var syncErr error
	resolved, syncErr = s.repo.Sync(ctx, requested)
	if syncErr != nil {
		resolved = requested
		fmt.Fprintf(&buf, "%s\n", syncErr)
		log.Printf("deploy %s failed: %v", requested, syncErr)
		return
	}
	last, err := s.store.LastPublished(ctx, "")
	if err != nil {
		fmt.Fprintf(&buf, "%s\n", err)
		return
	}
	if last == resolved {
		fmt.Fprintf(&buf, "already published\n")
		status = "published"
		return
	}
	fmt.Fprintf(&buf, "deploy %s\n", resolved)
	slugs, note, err := s.changedSlugs(ctx, last, resolved)
	if note != "" {
		fmt.Fprintf(&buf, "%s\n", note)
	}
	if err != nil {
		fmt.Fprintf(&buf, "%s\n", err)
		return
	}
	if len(slugs) == 0 {
		fmt.Fprintf(&buf, "apps: none\n")
	} else {
		fmt.Fprintf(&buf, "apps: %s\n", strings.Join(slugs, ", "))
		apps := make([]BuildApp, 0, len(slugs))
		for _, slug := range slugs {
			apps = append(apps, BuildApp{Slug: slug, Dir: "src/apps/" + slug})
		}
		buildLog, err := s.build(ctx, apps)
		buf.WriteString(buildLog)
		if buildLog != "" && !strings.HasSuffix(buildLog, "\n") {
			buf.WriteByte('\n')
		}
		if err != nil {
			fmt.Fprintf(&buf, "%s\n", err)
			return
		}
	}
	if err := Publish(s.cfg.SiteRoot, s.repo.Dir, s.outputDir(), slugs); err != nil {
		fmt.Fprintf(&buf, "%s\n", err)
		return
	}
	status = "published"
	fmt.Fprintf(&buf, "published\n")
}

func (s *Service) changedSlugs(ctx context.Context, last, sha string) ([]string, string, error) {
	apps, err := ListAppSlugs(filepath.Join(s.repo.Dir, "apps"))
	if err != nil {
		return nil, "", err
	}
	if ShouldBuildAll(last) {
		return apps, "", nil
	}
	diff, err := s.repo.DiffNames(ctx, last, sha)
	if err != nil {
		return apps, "diff failed, building every app: " + err.Error(), nil
	}
	return SlugsFromDiff(diff, apps), "", nil
}

func (s *Service) outputDir() string {
	return filepath.Join(s.cfg.Workspace, "out")
}

func capLog(text string) string {
	if len(text) <= maxLog {
		return text
	}
	return text[:maxLog]
}

func validRemote(remote string) bool {
	if remote == "" || strings.ContainsAny(remote, "\r\n\t ") || strings.HasPrefix(remote, "-") {
		return false
	}
	return true
}

func validBranch(branch string) bool {
	if branch == "" || strings.Contains(branch, "..") || strings.HasPrefix(branch, "/") || strings.HasSuffix(branch, "/") {
		return false
	}
	for _, r := range branch {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.' || r == '_' || r == '/' || r == '-':
		default:
			return false
		}
	}
	return true
}

func FromEnv(pool *pgxpool.Pool, siteRoot, publicBase string) (*Service, error) {
	var interval time.Duration
	if raw := strings.TrimSpace(os.Getenv("DEPLOY_POLL_INTERVAL")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return nil, fmt.Errorf("DEPLOY_POLL_INTERVAL: %w", err)
		}
		if parsed < 0 {
			return nil, fmt.Errorf("DEPLOY_POLL_INTERVAL must not be negative")
		}
		interval = parsed
	}
	return New(Config{
		Remote:        os.Getenv("GIT_REMOTE"),
		Branch:        os.Getenv("GIT_BRANCH"),
		WebhookSecret: os.Getenv("DEPLOY_WEBHOOK_SECRET"),
		DeployToken:   os.Getenv("DEPLOY_TOKEN"),
		PollInterval:  interval,
		BuilderURL:    os.Getenv("BUILDER_URL"),
		BuilderToken:  os.Getenv("BUILDER_TOKEN"),
		Workspace:     os.Getenv("WORKSPACE"),
		SiteRoot:      siteRoot,
		PublicBase:    publicBase,
		RuntimeDir:    os.Getenv("DEPLOY_RUNTIME_DIR"),
		Pool:          pool,
	})
}
