package api

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/jack-barr3tt/bouncer/internal/access"
	"github.com/jack-barr3tt/bouncer/internal/site"
	"github.com/jack-barr3tt/bouncer/internal/store"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

const cookieName = "bouncer_session"

var errHandled = errors.New("response handled")

type Config struct {
	Pool           *pgxpool.Pool
	SiteRoot       string
	HubDir         string
	PublicBaseURL  string
	BcryptCost     int
	TrustedProxies []string
	CookieSecure   string
}

type Server struct {
	store        *store.Store
	site         *site.Site
	broker       *access.Broker
	publicBase   string
	cookieSecure string
}

func NewApp(cfg Config) (*fiber.App, error) {
	opened, err := site.Open(cfg.SiteRoot, cfg.HubDir)
	if err != nil {
		return nil, err
	}
	st, err := store.New(cfg.Pool, cfg.BcryptCost)
	if err != nil {
		return nil, err
	}
	app := fiber.New(fiber.Config{
		TrustProxy: len(cfg.TrustedProxies) > 0,
		TrustProxyConfig: fiber.TrustProxyConfig{
			Proxies: cfg.TrustedProxies,
		},
		ProxyHeader: fiber.HeaderXForwardedFor,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			if errors.Is(err, errHandled) {
				return nil
			}
			return fiber.DefaultErrorHandler(c, err)
		},
	})
	srv := &Server{
		store:        st,
		site:         opened,
		broker:       access.New(),
		publicBase:   strings.TrimRight(cfg.PublicBaseURL, "/"),
		cookieSecure: strings.ToLower(strings.TrimSpace(cfg.CookieSecure)),
	}
	app.Use(srv.gate)
	RegisterHandlers(app, srv)
	return app, nil
}

func (s *Server) Login(c fiber.Ctx) error {
	var req LoginRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request.")
	}
	fp := strings.TrimSpace(stringValue(req.Fingerprint))
	buckets := loginBuckets(c.IP(), fp)
	locked, err := s.store.Locked(c.Context(), bucketNames(buckets))
	if err != nil {
		return s.internal(c, err)
	}
	if locked {
		return writeError(c, fiber.StatusTooManyRequests, "Too many attempts. Try again later.")
	}
	user, err := s.store.Authenticate(c.Context(), req.Username, req.Password)
	if errors.Is(err, store.ErrNotFound) {
		if err := s.store.Fail(c.Context(), buckets); err != nil {
			return s.internal(c, err)
		}
		return writeError(c, fiber.StatusUnauthorized, "Invalid username or password.")
	}
	if err != nil {
		return s.internal(c, err)
	}
	if err := s.store.Clear(c.Context(), bucketNames(buckets)); err != nil {
		return s.internal(c, err)
	}
	expires := time.Now().Add(store.UserSessionTTL)
	token, err := s.store.CreateSession(c.Context(), &user.ID, nil, expires)
	if err != nil {
		return s.internal(c, err)
	}
	s.setCookie(c, token, expires)
	principal, err := s.principal(c, token)
	if err != nil {
		return s.internal(c, err)
	}
	return c.JSON(sessionJSON(principal))
}

func (s *Server) Logout(c fiber.Ctx) error {
	token := c.Cookies(cookieName)
	if token != "" {
		if err := s.store.DeleteSessionToken(c.Context(), token); err != nil {
			return s.internal(c, err)
		}
	}
	s.clearCookie(c)
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) GetSession(c fiber.Ctx) error {
	principal, err := s.require(c)
	if err != nil {
		return err
	}
	if principal.TempID != nil {
		_ = s.store.TouchTempIP(c.Context(), *principal.TempID, c.IP())
	}
	return c.JSON(sessionJSON(principal))
}

func (s *Server) require(c fiber.Ctx) (*store.Principal, error) {
	principal, err := s.principal(c, c.Cookies(cookieName))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, writeError(c, fiber.StatusUnauthorized, "Sign in required.")
		}
		return nil, s.internal(c, err)
	}
	return principal, nil
}

func (s *Server) requireAdmin(c fiber.Ctx) (*store.Principal, error) {
	principal, err := s.require(c)
	if err != nil {
		return nil, err
	}
	if principal.Role != "admin" {
		return nil, writeError(c, fiber.StatusForbidden, "Admin only.")
	}
	return principal, nil
}

func (s *Server) principal(c fiber.Ctx, token string) (*store.Principal, error) {
	slugs, err := s.site.Slugs()
	if err != nil {
		return nil, err
	}
	return s.store.Principal(c.Context(), token, slugs)
}

func (s *Server) setCookie(c fiber.Ctx, token string, expires time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HTTPOnly: true,
		Secure:   s.secureCookie(c),
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}

func (s *Server) clearCookie(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   s.secureCookie(c),
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}

func (s *Server) secureCookie(c fiber.Ctx) bool {
	switch s.cookieSecure {
	case "true", "1":
		return true
	case "false", "0":
		return false
	}
	if c.Protocol() == "https" {
		return true
	}
	return c.IsProxyTrusted() && strings.EqualFold(c.Get(fiber.HeaderXForwardedProto), "https")
}

func (s *Server) internal(c fiber.Ctx, err error) error {
	log.Printf("request failed: %v", err)
	return writeError(c, fiber.StatusInternalServerError, "Something went wrong.")
}

func writeError(c fiber.Ctx, status int, message string) error {
	if err := c.Status(status).JSON(Error{Message: message}); err != nil {
		return err
	}
	return errHandled
}

func sessionJSON(p *store.Principal) Session {
	out := Session{
		Kind:      SessionKind(p.Kind),
		Apps:      p.Apps,
		ExpiresAt: p.ExpiresAt,
	}
	if out.Apps == nil {
		out.Apps = []string{}
	}
	if p.Kind == "user" {
		name := p.Username
		role := SessionRole(p.Role)
		out.Username = &name
		out.Role = &role
	} else {
		name := p.Nickname
		out.Nickname = &name
	}
	return out
}

func userJSON(user store.User) User {
	apps := user.Apps
	if apps == nil {
		apps = []string{}
	}
	return User{
		Id:       user.ID,
		Username: user.Username,
		Role:     UserRole(user.Role),
		Disabled: user.Disabled,
		Apps:     apps,
	}
}

func loginBuckets(ip, fingerprint string) []store.Bucket {
	threshold := 5
	if fingerprint == "" {
		threshold = 3
	}
	buckets := []store.Bucket{{Name: "login:ip:" + ip, Threshold: threshold}}
	if fingerprint != "" {
		buckets = append(buckets, store.Bucket{Name: "login:fp:" + store.HashFingerprint(fingerprint), Threshold: 5})
	}
	return buckets
}

func codeBuckets(ip, fingerprint string) []store.Bucket {
	buckets := []store.Bucket{{Name: "code:ip:" + ip, Threshold: 5}}
	if fingerprint != "" {
		buckets = append(buckets, store.Bucket{Name: "code:fp:" + store.HashFingerprint(fingerprint), Threshold: 5})
	}
	return buckets
}

func bucketNames(buckets []store.Bucket) []string {
	names := make([]string, len(buckets))
	for i, bucket := range buckets {
		names[i] = bucket.Name
	}
	return names
}

func (s *Server) knownSlugs(raw []string) ([]string, error) {
	seen := map[string]struct{}{}
	var slugs []string
	for _, slug := range raw {
		slug = strings.TrimSpace(slug)
		if slug == "" {
			continue
		}
		if _, ok := seen[slug]; ok {
			continue
		}
		if !s.site.Known(slug) {
			return nil, errors.New("unknown app")
		}
		seen[slug] = struct{}{}
		slugs = append(slugs, slug)
	}
	if slugs == nil {
		slugs = []string{}
	}
	return slugs, nil
}

func validUsername(raw string) (string, bool) {
	name := strings.TrimSpace(raw)
	if len(name) < 1 || len(name) > 64 {
		return "", false
	}
	for i, r := range name {
		letter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		digit := r >= '0' && r <= '9'
		if i == 0 && !letter && !digit {
			return "", false
		}
		if !letter && !digit && r != '.' && r != '_' && r != '-' {
			return "", false
		}
	}
	return name, true
}

func validPassword(password string) bool {
	return len(password) >= 8 && len(password) <= 200
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
