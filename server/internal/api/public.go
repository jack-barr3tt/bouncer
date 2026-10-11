package api

import (
	"context"

	"github.com/gofiber/fiber/v3"
)

func (s *Server) SetAppPublic(c fiber.Ctx, slug Slug) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	var req AppPublicRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request.")
	}
	ok, err := s.site.HasApp(slug)
	if err != nil {
		return s.internal(c, err)
	}
	if !ok {
		return writeError(c, fiber.StatusNotFound, "Unknown app.")
	}
	if err := s.store.SetPublic(c.Context(), slug, req.Public); err != nil {
		return s.internal(c, err)
	}
	if err := s.publishLiveAccess(c.Context()); err != nil {
		return s.internal(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) visibleApps(ctx context.Context, grants []string, role string) ([]string, error) {
	if role == "admin" {
		return s.site.Slugs()
	}
	public, err := s.knownPublic(ctx)
	if err != nil {
		return nil, err
	}
	return addSlugs(grants, public), nil
}

func (s *Server) knownPublic(ctx context.Context) ([]string, error) {
	public, err := s.store.PublicSlugs(ctx)
	if err != nil {
		return nil, err
	}
	known, err := s.site.Slugs()
	if err != nil {
		return nil, err
	}
	return keepSlugs(public, known), nil
}

func (s *Server) publishLiveAccess(ctx context.Context) error {
	sessions, err := s.store.LiveSessions(ctx)
	if err != nil {
		return err
	}
	known, err := s.site.Slugs()
	if err != nil {
		return err
	}
	public, err := s.knownPublic(ctx)
	if err != nil {
		return err
	}
	for _, session := range sessions {
		apps := addSlugs(session.Apps, public)
		if session.Role == "admin" {
			apps = known
		}
		s.broker.Publish(session.Key, accessEvent("access", apps))
	}
	return nil
}

func (s *Server) dropPublic(ctx context.Context, slugs []string) ([]string, error) {
	public, err := s.store.PublicSlugs(ctx)
	if err != nil {
		return nil, err
	}
	return dropSlugs(slugs, public), nil
}

func addSlugs(base, extra []string) []string {
	seen := map[string]struct{}{}
	out := append([]string{}, base...)
	for _, slug := range out {
		seen[slug] = struct{}{}
	}
	for _, slug := range extra {
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}
		out = append(out, slug)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func keepSlugs(have, allow []string) []string {
	set := map[string]struct{}{}
	for _, slug := range allow {
		set[slug] = struct{}{}
	}
	out := []string{}
	for _, slug := range have {
		if _, ok := set[slug]; ok {
			out = append(out, slug)
		}
	}
	return out
}

func dropSlugs(have, drop []string) []string {
	set := map[string]struct{}{}
	for _, slug := range drop {
		set[slug] = struct{}{}
	}
	out := []string{}
	for _, slug := range have {
		if _, ok := set[slug]; !ok {
			out = append(out, slug)
		}
	}
	return out
}

func publicSet(slugs []string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, slug := range slugs {
		set[slug] = struct{}{}
	}
	return set
}
