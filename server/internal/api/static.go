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
	kind, hostSlug, err := s.classifyHost(c)
	if err != nil {
		return s.internal(c, err)
	}
	if kind == hostUnknown {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if kind == hostApp {
		return s.serveApp(c, hostSlug, strings.TrimPrefix(requestPath, "/"), true)
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
		if kind == hostApex {
			return c.Redirect().Status(fiber.StatusFound).To(s.subdomainURL(c, slug, rel))
		}
		if rel == "" && !strings.HasSuffix(requestPath, "/") {
			return c.Redirect().Status(fiber.StatusFound).To("/apps/" + slug + "/")
		}
		return s.serveApp(c, slug, rel, false)
	}
	if trimmed == "apps.yaml" {
		return s.serveRegistry(c)
	}
	return s.serveHub(c, trimmed)
}

func (s *Server) subdomainURL(c fiber.Ctx, slug, rel string) string {
	target := s.route.origin(slug) + "/"
	if rel != "" {
		target += rel
	}
	if query := string(c.Request().URI().QueryString()); query != "" {
		target += "?" + query
	}
	return target
}

func (s *Server) serveApp(c fiber.Ctx, slug, rel string, onHost bool) error {
	principal, err := s.optionalPrincipal(c)
	if err != nil {
		return s.internal(c, err)
	}
	if principal == nil {
		if wantsHTML(c) {
			return c.Redirect().Status(fiber.StatusFound).To(s.loginLocation(c, slug, onHost))
		}
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	if !contains(principal.Apps, slug) {
		home := "/"
		if onHost && s.publicBase != "" {
			home = s.publicBase + "/"
		}
		c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
		return c.Status(fiber.StatusForbidden).SendString("<!doctype html><title>No access</title><p>You do not have access to this app.</p><p><a href=\"" + home + "\">All apps</a></p>")
	}
	app, ok, err := s.site.Lookup(slug)
	if err != nil {
		return s.internal(c, err)
	}
	if !ok {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if app.Upstream != "" {
		prefix := "/apps/" + slug
		if onHost {
			prefix = "/"
		}
		return s.proxyApp(c, app.Upstream, prefix)
	}
	return sendFile(c, s.site.AppDir(slug), rel)
}

func (s *Server) loginLocation(c fiber.Ctx, slug string, onHost bool) string {
	next := c.OriginalURL()
	if next == "" {
		next = c.Path()
	}
	if onHost {
		if !strings.HasPrefix(next, "/") {
			next = "/" + next
		}
		next = s.route.origin(slug) + next
	}
	return s.loginPath(next)
}

func (s *Server) loginPath(next string) string {
	loc := "/login?next=" + url.QueryEscape(s.safeNext(next))
	if strings.Contains(next, "://") {
		return s.publicBase + loc
	}
	return loc
}

func (s *Server) serveHub(c fiber.Ctx, rel string) error {
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
	return sendFile(c, s.site.HubDir(), "_shell.html")
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

func (s *Server) safeNext(raw string) string {
	if raw == "" || strings.HasPrefix(raw, "//") {
		return "/"
	}
	if strings.Contains(raw, "://") {
		return s.safeAppURL(raw)
	}
	if !strings.HasPrefix(raw, "/") {
		return "/"
	}
	return raw
}

func (s *Server) safeAppURL(raw string) string {
	if s.route.mode != "subdomain" {
		return "/"
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User != nil || parsed.Scheme != s.route.scheme || parsed.Hostname() == "" {
		return "/"
	}
	if parsed.Port() != s.route.port {
		return "/"
	}
	host := strings.ToLower(parsed.Hostname())
	suffix := "." + s.route.domain
	if !strings.HasSuffix(host, suffix) {
		return "/"
	}
	slug := strings.TrimSuffix(host, suffix)
	if slug == "" || strings.Contains(slug, ".") {
		return "/"
	}
	ok, err := s.site.HasApp(slug)
	if err != nil || !ok {
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
