package api

import (
	"path/filepath"

	"github.com/gofiber/fiber/v3"
	"github.com/jack-barr3tt/bouncer/internal/stack"
)

func (s *Server) GetUpdate(c fiber.Ctx) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	notice, ok, err := stack.ReadNotice(filepath.Join(s.site.Root(), "deploy", "update.json"))
	if err != nil {
		return s.internal(c, err)
	}
	if !ok {
		return c.JSON(UpdateNotice{Pending: false})
	}
	keys := notice.Keys
	return c.JSON(UpdateNotice{Pending: true, Version: &notice.Version, Keys: &keys})
}
