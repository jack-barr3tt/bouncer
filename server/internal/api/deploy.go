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

func (s *Server) CloneRepo(c fiber.Ctx) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	if s.deploy == nil {
		return writeError(c, fiber.StatusNotFound, "Deploy is not configured.")
	}
	var req CloneRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request.")
	}
	branch := ""
	if req.Branch != nil {
		branch = *req.Branch
	}
	cloned, err := s.deploy.AddClone(c.Context(), req.Remote, req.Name, branch)
	if err != nil {
		switch err.Error() {
		case "remote is not valid", "folder name is not valid", "branch is not valid":
			return writeError(c, fiber.StatusBadRequest, "Invalid request.")
		case "folder already exists":
			return writeError(c, fiber.StatusBadRequest, "That folder already exists.")
		case "could not clone that repository":
			return writeError(c, fiber.StatusBadRequest, "Could not clone that repository.")
		default:
			return s.internal(c, err)
		}
	}
	apps := make([]CloneApp, 0, len(cloned.Apps))
	for _, source := range cloned.Apps {
		apps = append(apps, CloneApp{Source: source})
	}
	return c.Status(fiber.StatusCreated).JSON(CloneResult{
		Path:   cloned.Path,
		Remote: cloned.Remote,
		Branch: cloned.Branch,
		Apps:   apps,
	})
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
