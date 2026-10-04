package api

import (
	"errors"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/jack-barr3tt/bouncer/internal/site"
	"github.com/jack-barr3tt/bouncer/internal/store"
)

var contentTypes = map[string]string{
	".html":  "text/html; charset=utf-8",
	".js":    "text/javascript; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".json":  "application/json; charset=utf-8",
	".svg":   "image/svg+xml",
	".png":   "image/png",
	".webp":  "image/webp",
	".ico":   "image/x-icon",
	".woff2": "font/woff2",
	".txt":   "text/plain; charset=utf-8",
	".yaml":  "application/yaml",
	".yml":   "application/yaml",
	".map":   "application/json; charset=utf-8",
}

func (s *Server) gate(c fiber.Ctx) error {
	if strings.HasPrefix(c.Path(), "/api/") || c.Path() == "/api" {
		return c.Next()
	}
	requestPath := c.Path()
	if decoded, err := url.PathUnescape(requestPath); err == nil {
		requestPath = decoded
	}
	if requestPath == "" || strings.Contains(requestPath, "..") || strings.Contains(requestPath, "\\") {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	trimmed := strings.Trim(requestPath, "/")
	if trimmed == "apps" {
		return c.Redirect().Status(fiber.StatusFound).To("/")
	}
	if rest, ok := strings.CutPrefix(trimmed, "apps/"); ok {
		slug, rel, _ := strings.Cut(rest, "/")
		ok, err := s.site.HasApp(slug)
		if err != nil {
			return s.internal(c, err)
		}
		if slug == "" || !ok {
			return c.SendStatus(fiber.StatusNotFound)
		}
		if rel == "" && !strings.HasSuffix(requestPath, "/") {
			return c.Redirect().Status(fiber.StatusFound).To("/apps/" + slug + "/")
		}
		return s.serveApp(c, slug, rel)
	}
	if trimmed == "apps.yaml" {
		return sendFile(c, s.site.Root(), "apps.yaml")
	}
	return s.serveHub(c, trimmed)
}

func (s *Server) serveApp(c fiber.Ctx, slug, rel string) error {
	principal, err := s.optionalPrincipal(c)
	if err != nil {
		return s.internal(c, err)
	}
	if principal == nil {
		if wantsHTML(c) {
			next := c.OriginalURL()
			if next == "" {
				next = c.Path()
			}
			return c.Redirect().Status(fiber.StatusFound).To("/login?next=" + url.QueryEscape(safeNext(next)))
		}
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	if !contains(principal.Apps, slug) {
		c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
		return c.Status(fiber.StatusForbidden).SendString("<!doctype html><title>No access</title><p>You do not have access to this app.</p><p><a href=\"/\">All apps</a></p>")
	}
	return sendFile(c, s.site.AppDir(slug), rel)
}

func (s *Server) serveHub(c fiber.Ctx, rel string) error {
	if rel == "login" || rel == "admin" || rel == "code" || strings.HasPrefix(rel, "admin/") || strings.HasPrefix(rel, "code/") {
		return sendFile(c, s.site.HubDir(), "index.html")
	}
	if rel != "" {
		full, ok := site.SafeJoin(s.site.HubDir(), rel)
		if !ok {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		info, err := os.Stat(full)
		if err == nil && !info.IsDir() {
			return sendFile(c, s.site.HubDir(), rel)
		}
	}
	if rel == "" || !strings.Contains(path.Base(rel), ".") {
		return sendFile(c, s.site.HubDir(), "index.html")
	}
	return c.SendStatus(fiber.StatusNotFound)
}

func (s *Server) optionalPrincipal(c fiber.Ctx) (*store.Principal, error) {
	principal, err := s.principal(c, c.Cookies(cookieName))
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil
	}
	return principal, err
}

func sendFile(c fiber.Ctx, dir, rel string) error {
	full, ok := site.SafeJoin(dir, rel)
	if !ok {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		return c.SendStatus(fiber.StatusNotFound)
	}
	ext := strings.ToLower(path.Ext(full))
	ctype := contentTypes[ext]
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	if ext == ".html" || ext == ".json" {
		c.Set(fiber.HeaderCacheControl, "no-store")
	}
	c.Set(fiber.HeaderContentType, ctype)
	return c.SendFile(full)
}

func wantsHTML(c fiber.Ctx) bool {
	switch c.Get("Sec-Fetch-Dest") {
	case "document", "iframe":
		return true
	case "":
		return strings.Contains(c.Get(fiber.HeaderAccept), "text/html")
	default:
		return false
	}
}

func safeNext(raw string) string {
	if raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.Contains(raw, "://") {
		return "/"
	}
	return raw
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
