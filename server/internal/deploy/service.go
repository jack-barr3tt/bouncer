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
}

type Service struct {
	cfg     Config
	store   history
	repo    Repo
	client  *http.Client
	execute func(context.Context, string)
	mu      sync.Mutex
	running bool
	active  string
	next    *string
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
	if cfg.Remote == "" {
		return &Service{cfg: cfg}, nil
	}
	if !validRemote(cfg.Remote) {
		return nil, fmt.Errorf("GIT_REMOTE is not valid")
	}
	if !validBranch(cfg.Branch) {
		return nil, fmt.Errorf("GIT_BRANCH is not valid")
	}
	if strings.TrimSpace(cfg.BuilderToken) == "" {
		return nil, fmt.Errorf("BUILDER_TOKEN is required when GIT_REMOTE is set")
	}
	if strings.Contains(cfg.RuntimeDir, "'") {
		return nil, fmt.Errorf("runtime dir is not valid")
	}
	if cfg.store == nil {
		if cfg.Pool == nil {
			return nil, fmt.Errorf("database is required when GIT_REMOTE is set")
		}
		cfg.store = &pgStore{pool: cfg.Pool}
	}
	svc := &Service{
		cfg:    cfg,
		store:  cfg.store,
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
	return svc, nil
}

func (s *Service) Enabled() bool { return s != nil && s.cfg.Remote != "" }

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
		Remote:     s.cfg.Remote,
		Branch:     s.cfg.Branch,
		WebhookURL: s.cfg.PublicBase + "/api/hooks/git",
	}
	if !out.Enabled || s.store == nil {
		if !out.Enabled {
			out.Remote = ""
			out.Branch = ""
			out.WebhookURL = ""
		}
		return out, nil
	}
	latest, err := s.store.Latest(ctx)
	if err != nil {
		return Status{}, err
	}
	out.Latest = latest
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
	tip, err := s.repo.Tip(ctx)
	if err != nil {
		return err
	}
	last, err := s.store.LastPublished(ctx)
	if err != nil {
		return err
	}
	if tip != last {
		s.Enqueue(tip)
	}
	return nil
}

func (s *Service) Enqueue(sha string) {
	if !s.Enabled() {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		if sha == s.active {
			return
		}
		copied := sha
		s.next = &copied
		return
	}
	s.running = true
	s.active = sha
	go s.loop(sha)
}

func (s *Service) loop(sha string) {
	for {
		s.execute(context.Background(), sha)
		s.mu.Lock()
		if s.next == nil {
			s.running = false
			s.active = ""
			s.mu.Unlock()
			return
		}
		sha = *s.next
		s.active = sha
		s.next = nil
		s.mu.Unlock()
	}
}

func (s *Service) run(ctx context.Context, sha string) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Hour)
	defer cancel()
	s.perform(ctx, sha)
}

func (s *Service) perform(ctx context.Context, requested string) {
	id, err := s.store.Start(ctx, requested)
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
	last, err := s.store.LastPublished(ctx)
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
		buildLog, err := s.build(ctx, slugs)
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
