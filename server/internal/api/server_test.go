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

func newApp(t *testing.T) *fiber.App {
	t.Helper()
	if pool == nil {
		if os.Getenv("CI") != "" || os.Getenv("FORGEJO_ACTIONS") != "" {
			t.Fatal("TEST_DATABASE_URL is required")
		}
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE auth_attempts, sessions, temporary_accounts, access_code_apps, access_codes, user_apps, users`); err != nil {
		t.Fatal(err)
	}
	accounts, err := store.New(pool, bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := accounts.EnsureAdmin(ctx, "admin", "password1"); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "apps.yaml"), "apps:\n  - slug: hello\n    name: Hello\n    description: Hi\n    path: /apps/hello/\n    icon: Hi\n")
	writeFile(t, filepath.Join(root, "apps", "hub", "dist", "index.html"), "hub")
	writeFile(t, filepath.Join(root, "apps", "hello", "dist", "index.html"), "hello")
	writeFile(t, filepath.Join(root, "apps", "hello", "dist", "assets", "app.js"), "js")
	app, err := api.NewApp(api.Config{
		Pool:          pool,
		SiteRoot:      root,
		PublicBaseURL: "http://127.0.0.1:8080",
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

func get(t *testing.T, app *fiber.App, path, cookieHeader string, document bool) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
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
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	return string(body)
}
