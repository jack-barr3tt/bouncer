package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jack-barr3tt/bouncer/db"
	"github.com/jack-barr3tt/bouncer/internal/api"
	"github.com/jack-barr3tt/bouncer/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		os.Exit(m.Run())
	}
	ctx := context.Background()
	var err error
	pool, err = db.Connect(ctx, url)
	if err != nil {
		panic(err)
	}
	if err := db.Migrate(ctx, pool); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestLoginGrantAndCode(t *testing.T) {
	app := newApp(t)
	login := postJSON(t, app, "/api/login", map[string]string{"username": "admin", "password": "password1"}, "")
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login: %d %s", login.StatusCode, readBody(login))
	}
	adminCookie := cookie(t, login)
	var session struct {
		Kind string   `json:"kind"`
		Role string   `json:"role"`
		Apps []string `json:"apps"`
	}
	decode(t, login, &session)
	if session.Kind != "user" || session.Role != "admin" || len(session.Apps) != 1 || session.Apps[0] != "hello" {
		t.Fatalf("session: %+v", session)
	}

	created := postJSON(t, app, "/api/users", map[string]string{"username": "sam", "password": "password2"}, adminCookie)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create user: %d %s", created.StatusCode, readBody(created))
	}
	var user struct {
		ID   string   `json:"id"`
		Apps []string `json:"apps"`
		Role string   `json:"role"`
	}
	decode(t, created, &user)
	if user.Role != "user" || len(user.Apps) != 0 {
		t.Fatalf("new user: %+v", user)
	}

	samLogin := postJSON(t, app, "/api/login", map[string]string{"username": "sam", "password": "password2"}, "")
	samCookie := cookie(t, samLogin)
	denied := get(t, app, "/apps/hello/", samCookie, true)
	if denied.StatusCode != http.StatusForbidden {
		t.Fatalf("ungranted app: %d", denied.StatusCode)
	}
	allowed := get(t, app, "/apps/hello/", adminCookie, true)
	if allowed.StatusCode != http.StatusOK || !strings.Contains(readBody(allowed), "hello") {
		t.Fatalf("admin app: %d %s", allowed.StatusCode, readBody(allowed))
	}

	guest := get(t, app, "/apps/hello/", "", true)
	if guest.StatusCode != http.StatusFound || !strings.HasPrefix(guest.Header.Get("Location"), "/login?next=") {
		t.Fatalf("redirect: %d %s", guest.StatusCode, guest.Header.Get("Location"))
	}
	asset := get(t, app, "/apps/hello/assets/app.js", "", false)
	if asset.StatusCode != http.StatusUnauthorized {
		t.Fatalf("asset: %d", asset.StatusCode)
	}
	page := get(t, app, "/code/ABCDEFGH", "", true)
	if page.StatusCode != http.StatusOK || !strings.Contains(readBody(page), "hub") {
		t.Fatalf("join page: %d %s", page.StatusCode, readBody(page))
	}

	grants := putJSON(t, app, "/api/users/"+user.ID+"/apps", map[string]any{"slugs": []string{"hello"}}, adminCookie)
	if grants.StatusCode != http.StatusOK {
		t.Fatalf("grants: %d %s", grants.StatusCode, readBody(grants))
	}
	opened := get(t, app, "/apps/hello/", samCookie, false)
	if opened.StatusCode != http.StatusOK {
		t.Fatalf("granted app: %d", opened.StatusCode)
	}

	expires := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	codeRes := postJSON(t, app, "/api/access-codes", map[string]any{
		"label": "visit", "appSlugs": []string{"hello"}, "expiresAt": expires, "maxSignups": 1,
	}, adminCookie)
	if codeRes.StatusCode != http.StatusCreated {
		t.Fatalf("code: %d %s", codeRes.StatusCode, readBody(codeRes))
	}
	var code struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	decode(t, codeRes, &code)
	if !strings.HasSuffix(code.URL, "/code/") && !strings.Contains(code.URL, "/code/") {
		t.Fatalf("url: %s", code.URL)
	}
	joinCode := code.URL[strings.LastIndex(code.URL, "/")+1:]

	first := postJSON(t, app, "/api/access-codes/redeem", map[string]string{
		"code": joinCode, "nickname": "Ada", "fingerprint": "browser-a",
	}, "")
	if first.StatusCode != http.StatusOK {
		t.Fatalf("redeem: %d %s", first.StatusCode, readBody(first))
	}
	var temp struct {
		Kind     string   `json:"kind"`
		Nickname string   `json:"nickname"`
		Apps     []string `json:"apps"`
	}
	decode(t, first, &temp)
	if temp.Kind != "temporary" || temp.Nickname != "Ada" || len(temp.Apps) != 1 {
		t.Fatalf("temp session: %+v", temp)
	}
	tempCookie := cookie(t, first)
	again := postJSON(t, app, "/api/access-codes/redeem", map[string]string{
		"code": joinCode, "nickname": "Other", "fingerprint": "browser-a",
	}, "")
	if again.StatusCode != http.StatusOK {
		t.Fatalf("resume: %d %s", again.StatusCode, readBody(again))
	}
	decode(t, again, &temp)
	if temp.Nickname != "Ada" {
		t.Fatalf("resume nickname: %+v", temp)
	}
	full := postJSON(t, app, "/api/access-codes/redeem", map[string]string{
		"code": joinCode, "nickname": "Bea", "fingerprint": "browser-b",
	}, "")
	if full.StatusCode != http.StatusBadRequest {
		t.Fatalf("full: %d %s", full.StatusCode, readBody(full))
	}

	accounts := get(t, app, "/api/access-codes/"+code.ID+"/accounts", adminCookie, false)
	var listed struct {
		Accounts []struct {
			ID       string `json:"id"`
			Nickname string `json:"nickname"`
		} `json:"accounts"`
	}
	decode(t, accounts, &listed)
	if len(listed.Accounts) != 1 || listed.Accounts[0].Nickname != "Ada" {
		t.Fatalf("accounts: %+v", listed.Accounts)
	}
	revoked := postJSON(t, app, "/api/temporary-accounts/"+listed.Accounts[0].ID+"/revoke", map[string]string{}, adminCookie)
	if revoked.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke temp: %d %s", revoked.StatusCode, readBody(revoked))
	}
	gone := get(t, app, "/api/session", tempCookie, false)
	if gone.StatusCode != http.StatusUnauthorized {
		t.Fatalf("revoked session: %d", gone.StatusCode)
	}
	blocked := postJSON(t, app, "/api/access-codes/redeem", map[string]string{
		"code": joinCode, "nickname": "Ada", "fingerprint": "browser-a",
	}, "")
	if blocked.StatusCode != http.StatusBadRequest {
		t.Fatalf("revoked fingerprint: %d %s", blocked.StatusCode, readBody(blocked))
	}

	qr := get(t, app, "/api/access-codes/"+code.ID+"/qr", adminCookie, false)
	if qr.StatusCode != http.StatusOK || qr.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("qr: %d %s", qr.StatusCode, qr.Header.Get("Content-Type"))
	}
}

func TestCodeRevokeLeavesExistingAccount(t *testing.T) {
	app := newApp(t)
	admin := cookie(t, postJSON(t, app, "/api/login", map[string]string{"username": "admin", "password": "password1"}, ""))
	expires := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	codeRes := postJSON(t, app, "/api/access-codes", map[string]any{
		"appSlugs": []string{"hello"}, "expiresAt": expires, "maxSignups": 2,
	}, admin)
	var code struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	decode(t, codeRes, &code)
	joinCode := code.URL[strings.LastIndex(code.URL, "/")+1:]
	joined := postJSON(t, app, "/api/access-codes/redeem", map[string]string{
		"code": joinCode, "nickname": "Ada", "fingerprint": "browser-a",
	}, "")
	tempCookie := cookie(t, joined)
	revoked := postJSON(t, app, "/api/access-codes/"+code.ID+"/revoke", map[string]string{}, admin)
	if revoked.StatusCode != http.StatusOK {
		t.Fatalf("revoke code: %d %s", revoked.StatusCode, readBody(revoked))
	}
	still := get(t, app, "/api/session", tempCookie, false)
	if still.StatusCode != http.StatusOK {
		t.Fatalf("existing account: %d %s", still.StatusCode, readBody(still))
	}
	fresh := postJSON(t, app, "/api/access-codes/redeem", map[string]string{
		"code": joinCode, "nickname": "Bea", "fingerprint": "browser-b",
	}, "")
	if fresh.StatusCode != http.StatusUnauthorized {
		t.Fatalf("new signup after revoke: %d %s", fresh.StatusCode, readBody(fresh))
	}
}

func TestLoginLockout(t *testing.T) {
	app := newApp(t)
	for i := 0; i < 3; i++ {
		res := postJSON(t, app, "/api/login", map[string]string{"username": "admin", "password": "wrong-password"}, "")
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("attempt %d: %d %s", i, res.StatusCode, readBody(res))
		}
	}
	locked := postJSON(t, app, "/api/login", map[string]string{"username": "admin", "password": "password1"}, "")
	if locked.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("lockout: %d %s", locked.StatusCode, readBody(locked))
	}
}

func TestDeployUnconfigured(t *testing.T) {
	app := newApp(t)
	hook := postJSON(t, app, "/api/hooks/git", map[string]string{"ref": "refs/heads/main"}, "")
	if hook.StatusCode != http.StatusNotFound {
		t.Fatalf("hook: %d %s", hook.StatusCode, readBody(hook))
	}
	trigger := postJSON(t, app, "/api/deploy", map[string]string{}, "")
	if trigger.StatusCode != http.StatusNotFound {
		t.Fatalf("trigger: %d %s", trigger.StatusCode, readBody(trigger))
	}
	guest := get(t, app, "/api/deploy", "", false)
	if guest.StatusCode != http.StatusUnauthorized {
		t.Fatalf("guest deploy: %d", guest.StatusCode)
	}
	admin := cookie(t, postJSON(t, app, "/api/login", map[string]string{"username": "admin", "password": "password1"}, ""))
	status := get(t, app, "/api/deploy", admin, false)
	body := readBody(status)
	if status.StatusCode != http.StatusOK || !strings.Contains(body, `"enabled":false`) {
		t.Fatalf("status: %d %s", status.StatusCode, body)
	}
	run := postJSON(t, app, "/api/deploy/run", map[string]string{}, admin)
	if run.StatusCode != http.StatusNotFound {
		t.Fatalf("run: %d %s", run.StatusCode, readBody(run))
	}
}

func TestProxyStripsTheAppPrefix(t *testing.T) {
	var hits atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path != "/items" || r.URL.RawQuery != "x=1" {
			t.Errorf("upstream path: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		if r.Host != "apps.example" {
			t.Errorf("host: %s", r.Host)
		}
		if r.Header.Get("X-Forwarded-Host") != "apps.example" {
			t.Errorf("forwarded host: %s", r.Header.Get("X-Forwarded-Host"))
		}
		if r.Header.Get("X-Forwarded-Prefix") != "/apps/notes" {
			t.Errorf("prefix: %s", r.Header.Get("X-Forwarded-Prefix"))
		}
		_, _ = w.Write([]byte("proxied"))
	}))
	t.Cleanup(upstream.Close)
	app := newConfiguredApp(t, configuredApp{registry: notesRegistry(upstream.URL)})
	admin := cookie(t, postJSON(t, app, "/api/login", map[string]string{"username": "admin", "password": "password1"}, ""))
	created := postJSON(t, app, "/api/users", map[string]string{"username": "sam", "password": "password2"}, admin)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create user: %d %s", created.StatusCode, readBody(created))
	}
	sam := cookie(t, postJSON(t, app, "/api/login", map[string]string{"username": "sam", "password": "password2"}, ""))
	denied := getHost(t, app, "/apps/notes/items?x=1", "apps.example", sam, false)
	if denied.StatusCode != http.StatusForbidden {
		t.Fatalf("ungranted: %d %s", denied.StatusCode, readBody(denied))
	}
	if hits.Load() != 0 {
		t.Fatalf("upstream was dialed without a grant: %d", hits.Load())
	}
	opened := getHost(t, app, "/apps/notes/items?x=1", "apps.example", admin, false)
	if opened.StatusCode != http.StatusOK || readBody(opened) != "proxied" {
		t.Fatalf("proxy: %d %s", opened.StatusCode, readBody(opened))
	}
	session := getHost(t, app, "/api/session", "apps.example", admin, false)
	if session.StatusCode != http.StatusOK || hits.Load() != 1 {
		t.Fatalf("api leaked to upstream: status %d hits %d", session.StatusCode, hits.Load())
	}
}

func TestProxyReportsADeadUpstream(t *testing.T) {
	app := newConfiguredApp(t, configuredApp{registry: notesRegistry("http://127.0.0.1:1")})
	admin := cookie(t, postJSON(t, app, "/api/login", map[string]string{"username": "admin", "password": "password1"}, ""))
	res := getHost(t, app, "/apps/notes/", "apps.example", admin, true)
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("dead upstream: %d %s", res.StatusCode, readBody(res))
	}
}

func TestSubdomainRouting(t *testing.T) {
	root := t.TempDir()
	app := newConfiguredApp(t, configuredApp{
		root:       root,
		publicBase: "https://example.com",
		routing:    "subdomain",
	})
	admin := cookie(t, postHost(t, app, "/api/login", "example.com", map[string]string{"username": "admin", "password": "password1"}))
	opened := getHost(t, app, "/", "hello.example.com", admin, true)
	if opened.StatusCode != http.StatusOK || readBody(opened) != "hello" {
		t.Fatalf("subdomain app: %d %s", opened.StatusCode, readBody(opened))
	}
	asset := getHost(t, app, "/assets/app.js", "hello.example.com", admin, false)
	if asset.StatusCode != http.StatusOK || readBody(asset) != "js" {
		t.Fatalf("subdomain asset: %d %s", asset.StatusCode, readBody(asset))
	}
	local := getHost(t, app, "/apps/hello/", "localhost", admin, true)
	if local.StatusCode != http.StatusOK || readBody(local) != "hello" {
		t.Fatalf("localhost path: %d %s", local.StatusCode, readBody(local))
	}
	guest := getHost(t, app, "/today", "hello.example.com", "", true)
	if guest.StatusCode != http.StatusFound || guest.Header.Get("Location") != "https://example.com/login?next=https%3A%2F%2Fhello.example.com%2Ftoday" {
		t.Fatalf("login redirect: %d %s", guest.StatusCode, guest.Header.Get("Location"))
	}
	moved := getHost(t, app, "/apps/hello/items?x=1", "example.com", "", true)
	if moved.StatusCode != http.StatusFound || moved.Header.Get("Location") != "https://hello.example.com/items?x=1" {
		t.Fatalf("apex redirect: %d %s", moved.StatusCode, moved.Header.Get("Location"))
	}
	registry := getHost(t, app, "/apps.yaml", "example.com", "", false)
	body := readBody(registry)
	if registry.StatusCode != http.StatusOK || !strings.Contains(body, "https://hello.example.com/") {
		t.Fatalf("served registry: %d %s", registry.StatusCode, body)
	}
	stored, err := os.ReadFile(filepath.Join(root, "apps.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(stored), "path: /apps/hello/") || strings.Contains(string(stored), "hello.example.com") {
		t.Fatalf("disk registry changed: %s", stored)
	}
}

func TestSessionCookieUsesThePublicDomain(t *testing.T) {
	app := newConfiguredApp(t, configuredApp{publicBase: "https://example.com", routing: "subdomain"})
	login := postHost(t, app, "/api/login", "example.com", map[string]string{"username": "admin", "password": "password1"})
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login: %d %s", login.StatusCode, readBody(login))
	}
	set := strings.ToLower(strings.Join(login.Header.Values("Set-Cookie"), "; "))
	if !strings.Contains(set, "domain=example.com") {
		t.Fatalf("cookie: %s", set)
	}
	local := newConfiguredApp(t, configuredApp{publicBase: "https://example.com", routing: "subdomain"})
	again := postHost(t, local, "/api/login", "localhost", map[string]string{"username": "admin", "password": "password1"})
	localCookie := strings.ToLower(strings.Join(again.Header.Values("Set-Cookie"), "; "))
	if strings.Contains(localCookie, "domain=") {
		t.Fatalf("localhost cookie: %s", localCookie)
	}
}

func notesRegistry(upstream string) string {
	return "apps:\n  - slug: notes\n    name: Notes\n    description: Hi\n    path: /apps/notes/\n    icon: N\n    upstream: " + upstream + "\n"
}

func TestLastAdmin(t *testing.T) {
	app := newApp(t)
	admin := cookie(t, postJSON(t, app, "/api/login", map[string]string{"username": "admin", "password": "password1"}, ""))
	list := get(t, app, "/api/users", admin, false)
	var users struct {
		Users []struct {
			ID string `json:"id"`
		} `json:"users"`
	}
	decode(t, list, &users)
	res := patchJSON(t, app, "/api/users/"+users.Users[0].ID, map[string]any{"disabled": true}, admin)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("disable last admin: %d %s", res.StatusCode, readBody(res))
	}
}

type configuredApp struct {
	registry   string
	publicBase string
	routing    string
	root       string
}

func newApp(t *testing.T) *fiber.App {
	t.Helper()
	return newConfiguredApp(t, configuredApp{})
}

func newConfiguredApp(t *testing.T, cfg configuredApp) *fiber.App {
	t.Helper()
	if pool == nil {
		if os.Getenv("CI") != "" || os.Getenv("FORGEJO_ACTIONS") != "" {
			t.Fatal("TEST_DATABASE_URL is required")
		}
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE deploys, auth_attempts, sessions, temporary_accounts, access_code_apps, access_codes, user_apps, users`); err != nil {
		t.Fatal(err)
	}
	accounts, err := store.New(pool, bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := accounts.EnsureAdmin(ctx, "admin", "password1"); err != nil {
		t.Fatal(err)
	}
	root := cfg.root
	if root == "" {
		root = t.TempDir()
	}
	registry := cfg.registry
	if registry == "" {
		registry = "apps:\n  - slug: hello\n    name: Hello\n    description: Hi\n    path: /apps/hello/\n    icon: Hi\n"
	}
	publicBase := cfg.publicBase
	if publicBase == "" {
		publicBase = "http://127.0.0.1:8080"
	}
	writeFile(t, filepath.Join(root, "apps.yaml"), registry)
	writeFile(t, filepath.Join(root, "apps", "hub", "dist", "_shell.html"), "hub")
	writeFile(t, filepath.Join(root, "apps", "hello", "dist", "index.html"), "hello")
	writeFile(t, filepath.Join(root, "apps", "hello", "dist", "assets", "app.js"), "js")
	app, err := api.NewApp(api.Config{
		Pool:          pool,
		SiteRoot:      root,
		PublicBaseURL: publicBase,
		Routing:       cfg.routing,
		BcryptCost:    bcrypt.MinCost,
		CookieSecure:  "false",
	})
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func postJSON(t *testing.T, app *fiber.App, path string, body any, cookieHeader string) *http.Response {
	t.Helper()
	return sendJSON(t, app, http.MethodPost, path, body, cookieHeader)
}

func putJSON(t *testing.T, app *fiber.App, path string, body any, cookieHeader string) *http.Response {
	t.Helper()
	return sendJSON(t, app, http.MethodPut, path, body, cookieHeader)
}

func patchJSON(t *testing.T, app *fiber.App, path string, body any, cookieHeader string) *http.Response {
	t.Helper()
	return sendJSON(t, app, http.MethodPatch, path, body, cookieHeader)
}

func sendJSON(t *testing.T, app *fiber.App, method, path string, body any, cookieHeader string) *http.Response {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if cookieHeader != "" {
		req.Header.Set("Cookie", cookieHeader)
	}
	res, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func getHost(t *testing.T, app *fiber.App, path, host, cookieHeader string, document bool) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Host = host
	return do(t, app, req, cookieHeader, document)
}

func postHost(t *testing.T, app *fiber.App, path, host string, body any) *http.Response {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Host = host
	req.Header.Set("Content-Type", "application/json")
	return do(t, app, req, "", false)
}

func get(t *testing.T, app *fiber.App, path, cookieHeader string, document bool) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	return do(t, app, req, cookieHeader, document)
}

func do(t *testing.T, app *fiber.App, req *http.Request, cookieHeader string, document bool) *http.Response {
	t.Helper()
	if cookieHeader != "" {
		req.Header.Set("Cookie", cookieHeader)
	}
	if document {
		req.Header.Set("Accept", "text/html")
		req.Header.Set("Sec-Fetch-Dest", "document")
	} else {
		req.Header.Set("Sec-Fetch-Dest", "empty")
	}
	res, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func cookie(t *testing.T, res *http.Response) string {
	t.Helper()
	for _, part := range res.Header.Values("Set-Cookie") {
		if strings.HasPrefix(part, "bouncer_session=") {
			return strings.Split(part, ";")[0]
		}
	}
	t.Fatalf("missing session cookie: %v", res.Header.Values("Set-Cookie"))
	return ""
}

func decode(t *testing.T, res *http.Response, dest any) {
	t.Helper()
	body := readBody(res)
	if err := json.Unmarshal([]byte(body), dest); err != nil {
		t.Fatalf("json: %v body %s", err, body)
	}
}

func readBody(res *http.Response) string {
	defer func() { _ = res.Body.Close() }()
	body, _ := io.ReadAll(res.Body)
	return string(body)
}
