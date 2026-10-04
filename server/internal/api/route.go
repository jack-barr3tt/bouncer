package api

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"
	"gopkg.in/yaml.v3"
)

type hostKind int

const (
	hostOther hostKind = iota
	hostApex
	hostApp
	hostUnknown
)

type publicRoute struct {
	mode   string
	scheme string
	domain string
	port   string
	base   string
}

func parseRouting(publicBase, routing string) (publicRoute, error) {
	routing = strings.ToLower(strings.TrimSpace(routing))
	if routing == "" {
		routing = "path"
	}
	if routing != "path" && routing != "subdomain" {
		return publicRoute{}, fmt.Errorf("ROUTING must be path or subdomain")
	}
	base := strings.TrimRight(strings.TrimSpace(publicBase), "/")
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Hostname() == "" {
		if routing == "subdomain" {
			return publicRoute{}, fmt.Errorf("PUBLIC_BASE_URL must be an absolute URL when ROUTING=subdomain")
		}
		return publicRoute{mode: "path", base: base}, nil
	}
	domain := strings.ToLower(parsed.Hostname())
	if routing == "subdomain" && localHost(domain) {
		log.Printf("ROUTING=subdomain ignored for %s; serving paths", domain)
		routing = "path"
	}
	return publicRoute{
		mode:   routing,
		scheme: parsed.Scheme,
		domain: domain,
		port:   parsed.Port(),
		base:   base,
	}, nil
}

func (r publicRoute) origin(slug string) string {
	host := slug + "." + r.domain
	if r.port != "" {
		host = net.JoinHostPort(host, r.port)
	}
	return r.scheme + "://" + host
}

func localHost(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	return net.ParseIP(host) != nil
}

func requestHost(c fiber.Ctx) string {
	return strings.TrimSuffix(strings.ToLower(c.Hostname()), ".")
}

func (s *Server) classifyHost(c fiber.Ctx) (hostKind, string, error) {
	if s.route.mode != "subdomain" {
		return hostOther, "", nil
	}
	host := requestHost(c)
	if host == "" || localHost(host) {
		return hostOther, "", nil
	}
	if host == s.route.domain {
		return hostApex, "", nil
	}
	suffix := "." + s.route.domain
	if !strings.HasSuffix(host, suffix) {
		return hostOther, "", nil
	}
	slug := strings.TrimSuffix(host, suffix)
	if slug == "" || strings.Contains(slug, ".") {
		return hostOther, "", nil
	}
	ok, err := s.site.HasApp(slug)
	if err != nil {
		return 0, "", err
	}
	if !ok {
		return hostUnknown, "", nil
	}
	return hostApp, slug, nil
}

func (s *Server) cookieDomain(c fiber.Ctx) string {
	if s.route.mode != "subdomain" {
		return ""
	}
	host := requestHost(c)
	if host == "" || localHost(host) {
		return ""
	}
	if host == s.route.domain || strings.HasSuffix(host, "."+s.route.domain) {
		return s.route.domain
	}
	return ""
}

func (s *Server) serveRegistry(c fiber.Ctx) error {
	kind, _, err := s.classifyHost(c)
	if err != nil {
		return s.internal(c, err)
	}
	if kind != hostApex {
		return sendFile(c, s.site.Root(), "apps.yaml")
	}
	apps, err := s.site.Apps()
	if err != nil {
		return s.internal(c, err)
	}
	type entry struct {
		Slug        string `yaml:"slug"`
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Path        string `yaml:"path"`
		Icon        string `yaml:"icon"`
		Source      string `yaml:"source,omitempty"`
		Upstream    string `yaml:"upstream,omitempty"`
	}
	out := struct {
		Apps []entry `yaml:"apps"`
	}{Apps: make([]entry, 0, len(apps))}
	for _, app := range apps {
		out.Apps = append(out.Apps, entry{
			Slug:        app.Slug,
			Name:        app.Name,
			Description: app.Description,
			Path:        s.route.origin(app.Slug) + "/",
			Icon:        app.Icon,
			Source:      app.Source,
			Upstream:    app.Upstream,
		})
	}
	body, err := yaml.Marshal(out)
	if err != nil {
		return s.internal(c, err)
	}
	c.Set(fiber.HeaderContentType, "application/yaml")
	return c.Send(body)
}
