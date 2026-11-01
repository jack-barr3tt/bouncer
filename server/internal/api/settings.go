package api

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v3"
	"github.com/jack-barr3tt/bouncer/internal/stack"
)

func (s *Server) GetSettings(c fiber.Ctx) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	body, err := os.ReadFile(s.deployEnvPath())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return s.internal(c, err)
	}
	return c.JSON(Settings{AutoUpdate: stack.AutoUpdate(stack.EnvValue(string(body), "AUTO_UPDATE"))})
}

func (s *Server) UpdateSettings(c fiber.Ctx) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	var req Settings
	if err := c.Bind().JSON(&req); err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request.")
	}
	path := s.deployEnvPath()
	body, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return s.internal(c, err)
	}
	value := "false"
	if req.AutoUpdate {
		value = "true"
	}
	next := stack.SetKey(string(body), "AUTO_UPDATE", value)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return s.internal(c, err)
	}
	if err := os.WriteFile(path, []byte(next), 0o644); err != nil {
		return s.internal(c, err)
	}
	if !req.AutoUpdate {
		err := os.Remove(filepath.Join(filepath.Dir(path), "update.json"))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return s.internal(c, err)
		}
	}
	return c.JSON(Settings{AutoUpdate: req.AutoUpdate})
}

func (s *Server) deployEnvPath() string {
	return filepath.Join(s.site.Root(), "deploy", ".env")
}
