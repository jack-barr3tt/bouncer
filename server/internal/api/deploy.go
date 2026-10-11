package api

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/jack-barr3tt/bouncer/internal/deploy"
	"github.com/jack-barr3tt/bouncer/internal/site"
)

func (s *Server) GitHook(c fiber.Ctx) error {
	if s.deploy == nil {
		return writeError(c, fiber.StatusNotFound, "Deploy is not configured.")
	}
	event := c.Get("X-GitHub-Event")
	if event == "" {
		event = c.Get("X-Gitea-Event")
	}
	status, remote, sha := s.deploy.DecideHook(event, c.Get("X-Hub-Signature-256"), append([]byte(nil), c.BodyRaw()...))
	if sha != "" {
		s.deploy.Enqueue(remote, sha)
	}
	switch status {
	case fiber.StatusAccepted, fiber.StatusNoContent:
		return c.SendStatus(status)
	case fiber.StatusUnauthorized:
		return writeError(c, status, "Invalid signature.")
	case fiber.StatusBadRequest:
		return writeError(c, status, "Invalid request.")
	default:
		return writeError(c, status, "Deploy is not configured.")
	}
}

func (s *Server) GetDeploy(c fiber.Ctx) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	if s.deploy == nil {
		return c.JSON(DeployStatus{Repos: []DeployRepo{}})
	}
	status, err := s.deploy.Status(c.Context())
	if err != nil {
		return s.internal(c, err)
	}
	return c.JSON(deployStatusJSON(status))
}

func (s *Server) TriggerDeploy(c fiber.Ctx) error {
	if s.deploy == nil || !s.deploy.TokenEnabled() {
		return writeError(c, fiber.StatusNotFound, "Deploy is not configured.")
	}
	if !s.deploy.AcceptBearer(c.Get("Authorization")) {
		return writeError(c, fiber.StatusUnauthorized, "Invalid token.")
	}
	remote, sha, err := deploy.ParseDeploy(c.Body())
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request.")
	}
	target, err := s.deploy.Target(remote)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, deployTargetMessage(err))
	}
	s.deploy.Enqueue(target, sha)
	return c.SendStatus(fiber.StatusAccepted)
}

func (s *Server) RunDeploy(c fiber.Ctx) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	if s.deploy == nil || !s.deploy.Enabled() {
		return writeError(c, fiber.StatusNotFound, "Deploy is not configured.")
	}
	remote, _, err := deploy.ParseDeploy(c.Body())
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request.")
	}
	target, err := s.deploy.Target(remote)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, deployTargetMessage(err))
	}
	s.deploy.Enqueue(target, "")
	return c.SendStatus(fiber.StatusAccepted)
}

func (s *Server) PreviewRepo(c fiber.Ctx) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	if s.deploy == nil {
		return writeError(c, fiber.StatusNotFound, "Deploy is not configured.")
	}
	var req RepoPreviewRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request.")
	}
	preview, err := s.deploy.PreviewRepo(c.Context(), req.Remote, req.Name, deref(req.Commit), deref(req.Branch))
	if err != nil {
		return repoError(c, s, err)
	}
	apps := make([]RepoAppDir, 0, len(preview.Apps))
	for _, app := range preview.Apps {
		apps = append(apps, RepoAppDir{Dir: app.Dir})
	}
	return c.JSON(RepoPreview{
		Name:   preview.Name,
		Remote: preview.Remote,
		Commit: emptyNil(preview.Commit),
		Branch: emptyNil(preview.Branch),
		Apps:   apps,
	})
}

func (s *Server) AddRepo(c fiber.Ctx) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	var req AddRepoRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request.")
	}
	apps := make([]site.App, 0, len(req.Apps))
	for _, app := range req.Apps {
		apps = append(apps, site.App{
			Slug:        app.Slug,
			Name:        app.Name,
			Description: app.Description,
			Icon:        app.Icon,
			Dir:         app.Dir,
		})
	}
	err := s.site.AddRepo(site.RepoRef{
		Name:   req.Name,
		Remote: req.Remote,
		Commit: deref(req.Commit),
		Branch: deref(req.Branch),
	}, apps)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, repoWriteMessage(err))
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func emptyNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func repoError(c fiber.Ctx, s *Server, err error) error {
	message := err.Error()
	switch {
	case strings.Contains(message, "not valid"), strings.Contains(message, "set a branch or a commit"):
		return writeError(c, fiber.StatusBadRequest, "Invalid request.")
	case strings.Contains(message, "could not fetch"):
		return writeError(c, fiber.StatusBadRequest, "Could not fetch that repository.")
	default:
		return s.internal(c, err)
	}
}

func repoWriteMessage(err error) string {
	switch {
	case strings.Contains(err.Error(), "already registered"):
		return "That name is already used."
	case strings.Contains(err.Error(), "set a branch or a commit"):
		return "Set a branch or a commit."
	default:
		return "Invalid request."
	}
}

func (s *Server) CreateApp(c fiber.Ctx) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	var req CreateAppRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request.")
	}
	err := s.site.AddApp(site.App{
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		Source:      req.Source,
	})
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, appWriteMessage(err))
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func appWriteMessage(err error) string {
	switch {
	case strings.Contains(err.Error(), "already registered"):
		return "That slug is already used."
	case strings.Contains(err.Error(), "package.json"):
		return "That directory has no package.json."
	default:
		return "Invalid request."
	}
}

func deployTargetMessage(err error) string {
	switch err.Error() {
	case "remote is required":
		return "Remote is required."
	case "unknown remote":
		return "Unknown remote."
	default:
		return "Invalid request."
	}
}

func deployStatusJSON(status deploy.Status) DeployStatus {
	out := DeployStatus{
		Enabled:    status.Enabled,
		Remote:     status.Remote,
		Branch:     status.Branch,
		WebhookUrl: status.WebhookURL,
		Repos:      []DeployRepo{},
	}
	if status.Latest != nil {
		run := deployRunJSON(*status.Latest)
		out.Latest = &run
	}
	for _, repo := range status.Repos {
		item := DeployRepo{
			Path:   repo.Path,
			Remote: repo.Remote,
			Branch: repo.Branch,
			Commit: emptyNil(repo.Commit),
		}
		if repo.Latest != nil {
			run := deployRunJSON(*repo.Latest)
			item.Latest = &run
		}
		out.Repos = append(out.Repos, item)
	}
	return out
}

func deployRunJSON(run deploy.Run) DeployRun {
	return DeployRun{
		Sha:        run.SHA,
		Status:     DeployRunStatus(run.Status),
		Log:        run.Log,
		StartedAt:  run.StartedAt,
		FinishedAt: run.FinishedAt,
	}
}
