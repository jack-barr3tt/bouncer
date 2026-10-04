package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/jack-barr3tt/bouncer/internal/deploy"
)

func (s *Server) GitHook(c fiber.Ctx) error {
	if s.deploy == nil {
		return writeError(c, fiber.StatusNotFound, "Deploy is not configured.")
	}
	event := c.Get("X-GitHub-Event")
	if event == "" {
		event = c.Get("X-Gitea-Event")
	}
	status, sha := s.deploy.DecideHook(event, c.Get("X-Hub-Signature-256"), append([]byte(nil), c.BodyRaw()...))
	if sha != "" {
		s.deploy.Enqueue(sha)
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
		return c.JSON(DeployStatus{})
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
	sha, err := deploy.ParseDeploySHA(c.Body())
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request.")
	}
	s.deploy.Enqueue(sha)
	return c.SendStatus(fiber.StatusAccepted)
}

func (s *Server) RunDeploy(c fiber.Ctx) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	if s.deploy == nil || !s.deploy.Enabled() {
		return writeError(c, fiber.StatusNotFound, "Deploy is not configured.")
	}
	s.deploy.Enqueue("")
	return c.SendStatus(fiber.StatusAccepted)
}

func deployStatusJSON(status deploy.Status) DeployStatus {
	out := DeployStatus{
		Enabled:    status.Enabled,
		Remote:     status.Remote,
		Branch:     status.Branch,
		WebhookUrl: status.WebhookURL,
	}
	if status.Latest == nil {
		return out
	}
	out.Latest = &DeployRun{
		Sha:        status.Latest.SHA,
		Status:     DeployRunStatus(status.Latest.Status),
		Log:        status.Latest.Log,
		StartedAt:  status.Latest.StartedAt,
		FinishedAt: status.Latest.FinishedAt,
	}
	return out
}
