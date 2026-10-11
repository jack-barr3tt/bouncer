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
	if denied.StatusCode != http.StatusFound || denied.Header.Get("Location") != "/" {
		t.Fatalf("ungranted app: %d %s", denied.StatusCode, denied.Header.Get("Location"))
	}
	hidden := get(t, app, "/api/apps", samCookie, false)
	hiddenBody := readBody(hidden)
	if hidden.StatusCode != http.StatusOK || strings.Contains(hiddenBody, "hello") {
		t.Fatalf("ungranted catalog: %d %s", hidden.StatusCode, hiddenBody)
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

	grants := putJSON(t, app, "/api/users/"+user.ID+"/apps", map[string]any{"grants": []map[string]any{{"slug": "hello", "canShare": false}}}, adminCookie)
	if grants.StatusCode != http.StatusOK {
		t.Fatalf("grants: %d %s", grants.StatusCode, readBody(grants))
	}
	opened := get(t, app, "/apps/hello/", samCookie, false)
	if opened.StatusCode != http.StatusOK {
		t.Fatalf("granted app: %d", opened.StatusCode)
	}
	catalog := get(t, app, "/api/apps", samCookie, false)
	catalogBody := readBody(catalog)
	if catalog.StatusCode != http.StatusOK || !strings.Contains(catalogBody, "hello") {
		t.Fatalf("granted catalog: %d %s", catalog.StatusCode, catalogBody)
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

func TestDeleteAccessCodeEndsSessions(t *testing.T) {
	app := newApp(t)
	admin := cookie(t, postJSON(t, app, "/api/login", map[string]string{"username": "admin", "password": "password1"}, ""))
	expires := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	codeRes := postJSON(t, app, "/api/access-codes", map[string]any{
		"label": "gone", "appSlugs": []string{"hello"}, "expiresAt": expires, "maxSignups": 2,
	}, admin)
	if codeRes.StatusCode != http.StatusCreated {
		t.Fatalf("code: %d %s", codeRes.StatusCode, readBody(codeRes))
	}
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

	missing := deleteReq(t, app, "/api/access-codes/"+code.ID, "")
	if missing.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous delete: %d", missing.StatusCode)
	}
	deleted := deleteReq(t, app, "/api/access-codes/"+code.ID, admin)
	if deleted.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: %d %s", deleted.StatusCode, readBody(deleted))
	}
	gone := get(t, app, "/api/session", tempCookie, false)
	if gone.StatusCode != http.StatusUnauthorized {
		t.Fatalf("deleted session: %d", gone.StatusCode)
	}
	again := postJSON(t, app, "/api/access-codes/redeem", map[string]string{
		"code": joinCode, "nickname": "Bea", "fingerprint": "browser-b",
	}, "")
	if again.StatusCode != http.StatusUnauthorized {
		t.Fatalf("redeem deleted: %d %s", again.StatusCode, readBody(again))
	}
	listed := get(t, app, "/api/access-codes", admin, false)
	if strings.Contains(readBody(listed), code.ID) {
		t.Fatalf("code still listed: %s", readBody(listed))
	}
	unknown := deleteReq(t, app, "/api/access-codes/"+code.ID, admin)
	if unknown.StatusCode != http.StatusNotFound {
		t.Fatalf("second delete: %d %s", unknown.StatusCode, readBody(unknown))
	}
}

func TestUserSharesOwnedCodes(t *testing.T) {
	app := newConfiguredApp(t, configuredApp{registry: "apps:\n  - slug: hello\n    name: Hello\n    description: Hi\n    path: /apps/hello/\n    icon: Hi\n  - slug: notes\n    name: Notes\n    description: Hi\n    path: /apps/notes/\n    icon: N\n"})
	admin := cookie(t, postJSON(t, app, "/api/login", map[string]string{"username": "admin", "password": "password1"}, ""))
	created := postJSON(t, app, "/api/users", map[string]string{"username": "sam", "password": "password2"}, admin)
	var sam struct {
		ID        string   `json:"id"`
		ShareApps []string `json:"shareApps"`
	}
	decode(t, created, &sam)
	if len(sam.ShareApps) != 0 {
		t.Fatal("new account can share")
	}
	putJSON(t, app, "/api/users/"+sam.ID+"/apps", map[string]any{"grants": []map[string]any{{"slug": "hello", "canShare": false}}}, admin)
	samCookie := cookie(t, postJSON(t, app, "/api/login", map[string]string{"username": "sam", "password": "password2"}, ""))
	expires := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)

	blocked := postJSON(t, app, "/api/access-codes", map[string]any{
		"appSlugs": []string{"hello"}, "expiresAt": expires, "maxSignups": 1,
	}, samCookie)
	if blocked.StatusCode != http.StatusForbidden {
		t.Fatalf("share off: %d %s", blocked.StatusCode, readBody(blocked))
	}

	adminCode := postJSON(t, app, "/api/access-codes", map[string]any{
		"label": "admin", "appSlugs": []string{"notes"}, "expiresAt": expires, "maxSignups": 1,
	}, admin)
	var owned struct {
		ID    string `json:"id"`
		Owner string `json:"owner"`
	}
	decode(t, adminCode, &owned)
	if owned.Owner != "admin" {
		t.Fatalf("admin owner: %+v", owned)
	}
	if postJSON(t, app, "/api/access-codes/"+owned.ID+"/revoke", map[string]string{}, samCookie).StatusCode != http.StatusNotFound {
		t.Fatal("user revoked an admin code")
	}
	listed := readBody(get(t, app, "/api/access-codes", samCookie, false))
	if strings.Contains(listed, owned.ID) {
		t.Fatalf("user listed someone else's code: %s", listed)
	}

	shared := putJSON(t, app, "/api/users/"+sam.ID+"/apps", map[string]any{"grants": []map[string]any{
		{"slug": "hello", "canShare": true},
		{"slug": "notes", "canShare": false},
	}}, admin)
	if shared.StatusCode != http.StatusOK || !strings.Contains(readBody(shared), `"shareApps":["hello"]`) {
		t.Fatalf("enable share: %d %s", shared.StatusCode, readBody(shared))
	}
	session := get(t, app, "/api/session", samCookie, false)
	if !strings.Contains(readBody(session), `"shareApps":["hello"]`) {
		t.Fatalf("session share: %s", readBody(session))
	}
	outside := postJSON(t, app, "/api/access-codes", map[string]any{
		"appSlugs": []string{"notes"}, "expiresAt": expires, "maxSignups": 1,
	}, samCookie)
	if outside.StatusCode != http.StatusForbidden {
		t.Fatalf("ungranted app: %d %s", outside.StatusCode, readBody(outside))
	}

	made := postJSON(t, app, "/api/access-codes", map[string]any{
		"label": "visit", "appSlugs": []string{"hello"}, "expiresAt": expires, "maxSignups": 2,
	}, samCookie)
	if made.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d %s", made.StatusCode, readBody(made))
	}
	var code struct {
		ID    string `json:"id"`
		Owner string `json:"owner"`
		URL   string `json:"url"`
	}
	decode(t, made, &code)
	if code.Owner != "sam" {
		t.Fatalf("owner: %+v", code)
	}
	joinCode := code.URL[strings.LastIndex(code.URL, "/")+1:]
	joined := postJSON(t, app, "/api/access-codes/redeem", map[string]string{
		"code": joinCode, "nickname": "Ada", "fingerprint": "browser-a",
	}, "")
	var accounts struct {
		Accounts []struct {
			ID string `json:"id"`
		} `json:"accounts"`
	}
	decode(t, get(t, app, "/api/access-codes/"+code.ID+"/accounts", samCookie, false), &accounts)
	if len(accounts.Accounts) != 1 {
		t.Fatalf("accounts: %+v", accounts.Accounts)
	}

	lea := postJSON(t, app, "/api/users", map[string]string{"username": "lea", "password": "password3"}, admin)
	var other struct {
		ID string `json:"id"`
	}
	decode(t, lea, &other)
	putJSON(t, app, "/api/users/"+other.ID+"/apps", map[string]any{"grants": []map[string]any{{"slug": "hello", "canShare": true}}}, admin)
	leaCookie := cookie(t, postJSON(t, app, "/api/login", map[string]string{"username": "lea", "password": "password3"}, ""))
	if postJSON(t, app, "/api/access-codes/"+code.ID+"/revoke", map[string]string{}, leaCookie).StatusCode != http.StatusNotFound {
		t.Fatal("another user revoked the code")
	}
	if postJSON(t, app, "/api/temporary-accounts/"+accounts.Accounts[0].ID+"/revoke", map[string]string{}, leaCookie).StatusCode != http.StatusNotFound {
		t.Fatal("another user revoked the guest")
	}

	revokedGuest := postJSON(t, app, "/api/temporary-accounts/"+accounts.Accounts[0].ID+"/revoke", map[string]string{}, samCookie)
	if revokedGuest.StatusCode != http.StatusNoContent {
		t.Fatalf("owner revoke guest: %d %s", revokedGuest.StatusCode, readBody(revokedGuest))
	}
	if get(t, app, "/api/session", cookie(t, joined), false).StatusCode != http.StatusUnauthorized {
		t.Fatal("guest stayed signed in")
	}

	byAdmin := postJSON(t, app, "/api/access-codes/"+code.ID+"/revoke", map[string]string{}, admin)
	if byAdmin.StatusCode != http.StatusOK {
		t.Fatalf("admin revoke: %d %s", byAdmin.StatusCode, readBody(byAdmin))
	}
	again := postJSON(t, app, "/api/access-codes", map[string]any{
		"appSlugs": []string{"hello"}, "expiresAt": expires, "maxSignups": 1,
	}, samCookie)
	var second struct {
		ID string `json:"id"`
	}
	decode(t, again, &second)
	if deleteReq(t, app, "/api/access-codes/"+second.ID, samCookie).StatusCode != http.StatusNoContent {
		t.Fatal("owner could not delete")
	}
	if postJSON(t, app, "/api/access-codes/redeem", map[string]string{
		"code": joinCode, "nickname": "Bea", "fingerprint": "browser-b",
	}, "").StatusCode != http.StatusUnauthorized {
		t.Fatal("revoked code still joined")
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

func TestHubUpstreamProxiesTheHomepage(t *testing.T) {
	var hits atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path != "/login" {
			t.Errorf("hub path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte("signed-in page"))
	}))
	t.Cleanup(upstream.Close)
	app := newConfiguredApp(t, configuredApp{hubUpstream: upstream.URL})
	page := get(t, app, "/login", "", true)
	if page.StatusCode != http.StatusOK || readBody(page) != "signed-in page" {
		t.Fatalf("hub: %d %s", page.StatusCode, readBody(page))
	}
	session := get(t, app, "/api/session", "", false)
	if session.StatusCode == http.StatusOK || hits.Load() != 1 {
		t.Fatalf("api leaked to the hub: status %d hits %d", session.StatusCode, hits.Load())
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
	guestCatalog := getHost(t, app, "/api/apps", "example.com", "", false)
	guestBody := readBody(guestCatalog)
	if guestCatalog.StatusCode != http.StatusOK || strings.Contains(guestBody, "hello") {
		t.Fatalf("guest catalog: %d %s", guestCatalog.StatusCode, guestBody)
	}
	registry := getHost(t, app, "/api/apps", "example.com", admin, false)
	body := readBody(registry)
	if registry.StatusCode != http.StatusOK || !strings.Contains(body, "https://hello.example.com/") {
		t.Fatalf("app catalog: %d %s", registry.StatusCode, body)
	}
	file := getHost(t, app, "/apps.yaml", "example.com", "", false)
	if strings.Contains(readBody(file), "path:") || strings.Contains(readBody(file), "slug:") {
		t.Fatalf("yaml was served: %s", readBody(file))
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

func TestPublicApps(t *testing.T) {
	root := t.TempDir()
	registry := "" +
		"apps:\n" +
		"  - slug: hello\n    name: Hello\n    description: Hi\n    path: /apps/hello/\n    icon: Hi\n" +
		"  - slug: notes\n    name: Notes\n    description: Notes\n    path: /apps/notes/\n    icon: N\n"
	app := newConfiguredApp(t, configuredApp{root: root, registry: registry})
	writeFile(t, filepath.Join(root, "apps", "notes", "dist", "index.html"), "notes")

	admin := cookie(t, postJSON(t, app, "/api/login", map[string]string{"username": "admin", "password": "password1"}, ""))
	created := postJSON(t, app, "/api/users", map[string]string{"username": "sam", "password": "password2"}, admin)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create user: %d %s", created.StatusCode, readBody(created))
	}
	var user struct {
		ID string `json:"id"`
	}
	decode(t, created, &user)
	sam := cookie(t, postJSON(t, app, "/api/login", map[string]string{"username": "sam", "password": "password2"}, ""))
	if created := postJSON(t, app, "/api/users", map[string]string{"username": "lee", "password": "password3"}, admin); created.StatusCode != http.StatusCreated {
		t.Fatalf("create lee: %d %s", created.StatusCode, readBody(created))
	}
	lee := cookie(t, postJSON(t, app, "/api/login", map[string]string{"username": "lee", "password": "password3"}, ""))

	hidden := readBody(get(t, app, "/api/apps", "", false))
	if strings.Contains(hidden, "hello") || strings.Contains(hidden, "notes") {
		t.Fatalf("signed-out catalog: %s", hidden)
	}
	if guest := get(t, app, "/apps/hello/", "", true); guest.StatusCode != http.StatusFound {
		t.Fatalf("private redirect: %d", guest.StatusCode)
	}

	if res := patchJSON(t, app, "/api/apps/hello", map[string]bool{"public": true}, sam); res.StatusCode != http.StatusForbidden {
		t.Fatalf("user visibility: %d", res.StatusCode)
	}
	if res := patchJSON(t, app, "/api/apps/missing", map[string]bool{"public": true}, admin); res.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown app: %d %s", res.StatusCode, readBody(res))
	}
	if res := patchJSON(t, app, "/api/apps/hello", map[string]bool{"public": true}, admin); res.StatusCode != http.StatusNoContent {
		t.Fatalf("make public: %d %s", res.StatusCode, readBody(res))
	}

	var catalog struct {
		Apps []struct {
			Slug   string `json:"slug"`
			Public bool   `json:"public"`
		} `json:"apps"`
	}
	decode(t, get(t, app, "/api/apps", "", false), &catalog)
	if len(catalog.Apps) != 1 || catalog.Apps[0].Slug != "hello" || !catalog.Apps[0].Public {
		t.Fatalf("public catalog: %+v", catalog.Apps)
	}
	if opened := get(t, app, "/apps/hello/", "", true); opened.StatusCode != http.StatusOK || !strings.Contains(readBody(opened), "hello") {
		t.Fatalf("public html: %d", opened.StatusCode)
	}
	if asset := get(t, app, "/apps/hello/assets/app.js", "", false); asset.StatusCode != http.StatusOK {
		t.Fatalf("public asset: %d", asset.StatusCode)
	}
	if notes := get(t, app, "/apps/notes/", "", true); notes.StatusCode != http.StatusFound {
		t.Fatalf("private notes: %d", notes.StatusCode)
	}
	if opened := get(t, app, "/apps/hello/", sam, false); opened.StatusCode != http.StatusOK {
		t.Fatalf("user public app: %d", opened.StatusCode)
	}
	if opened := get(t, app, "/apps/hello/", lee, false); opened.StatusCode != http.StatusOK {
		t.Fatalf("ungranted public app: %d", opened.StatusCode)
	}
	if denied := get(t, app, "/apps/notes/", sam, false); denied.StatusCode != http.StatusForbidden {
		t.Fatalf("user private app: %d", denied.StatusCode)
	}
	decode(t, get(t, app, "/api/apps", sam, false), &catalog)
	if len(catalog.Apps) != 1 || catalog.Apps[0].Slug != "hello" {
		t.Fatalf("user catalog: %+v", catalog.Apps)
	}

	grants := putJSON(t, app, "/api/users/"+user.ID+"/apps", map[string]any{
		"grants": []map[string]any{{"slug": "hello", "canShare": true}, {"slug": "notes", "canShare": false}},
	}, admin)
	if grants.StatusCode != http.StatusOK {
		t.Fatalf("grants: %d %s", grants.StatusCode, readBody(grants))
	}
	var listed struct {
		Users []struct {
			ID   string   `json:"id"`
			Apps []string `json:"apps"`
		} `json:"users"`
	}
	decode(t, get(t, app, "/api/users", admin, false), &listed)
	var stored []string
	for _, item := range listed.Users {
		if item.ID == user.ID {
			stored = item.Apps
		}
	}
	if len(stored) != 2 {
		t.Fatalf("grants kept while public: %+v", stored)
	}

	expires := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	onlyPublic := postJSON(t, app, "/api/access-codes", map[string]any{
		"label": "public", "appSlugs": []string{"hello"}, "expiresAt": expires, "maxSignups": 1,
	}, admin)
	if onlyPublic.StatusCode != http.StatusBadRequest {
		t.Fatalf("public-only code: %d %s", onlyPublic.StatusCode, readBody(onlyPublic))
	}
	codeRes := postJSON(t, app, "/api/access-codes", map[string]any{
		"label": "visit", "appSlugs": []string{"hello", "notes"}, "expiresAt": expires, "maxSignups": 1,
	}, admin)
	if codeRes.StatusCode != http.StatusCreated {
		t.Fatalf("code: %d %s", codeRes.StatusCode, readBody(codeRes))
	}
	var code struct {
		URL  string   `json:"url"`
		Apps []string `json:"apps"`
	}
	decode(t, codeRes, &code)
	if len(code.Apps) != 1 || code.Apps[0] != "notes" {
		t.Fatalf("code apps: %+v", code.Apps)
	}
	joinCode := code.URL[strings.LastIndex(code.URL, "/")+1:]
	redeemed := postJSON(t, app, "/api/access-codes/redeem", map[string]string{
		"code": joinCode, "nickname": "Ada", "fingerprint": "browser-public",
	}, "")
	if redeemed.StatusCode != http.StatusOK {
		t.Fatalf("redeem: %d %s", redeemed.StatusCode, readBody(redeemed))
	}
	var temp struct {
		Apps []string `json:"apps"`
	}
	decode(t, redeemed, &temp)
	if !containsAll(temp.Apps, "hello", "notes") {
		t.Fatalf("temp apps: %+v", temp.Apps)
	}
	tempCookie := cookie(t, redeemed)
	if opened := get(t, app, "/apps/hello/", tempCookie, false); opened.StatusCode != http.StatusOK {
		t.Fatalf("temp public app: %d", opened.StatusCode)
	}

	if res := patchJSON(t, app, "/api/apps/hello", map[string]bool{"public": false}, admin); res.StatusCode != http.StatusNoContent {
		t.Fatalf("make private: %d %s", res.StatusCode, readBody(res))
	}
	if guest := get(t, app, "/apps/hello/", "", true); guest.StatusCode != http.StatusFound {
		t.Fatalf("private again: %d", guest.StatusCode)
	}
	if opened := get(t, app, "/apps/hello/", sam, false); opened.StatusCode != http.StatusOK {
		t.Fatalf("grant restored: %d", opened.StatusCode)
	}
	if denied := get(t, app, "/apps/hello/", lee, false); denied.StatusCode != http.StatusForbidden {
		t.Fatalf("ungranted lost public app: %d", denied.StatusCode)
	}
	decode(t, get(t, app, "/api/session", tempCookie, false), &temp)
	if len(temp.Apps) != 1 || temp.Apps[0] != "notes" {
		t.Fatalf("temp after private: %+v", temp.Apps)
	}
	if denied := get(t, app, "/apps/hello/", tempCookie, false); denied.StatusCode != http.StatusForbidden {
		t.Fatalf("temp lost public app: %d", denied.StatusCode)
	}
}

func containsAll(have []string, want ...string) bool {
	set := map[string]struct{}{}
	for _, slug := range have {
		set[slug] = struct{}{}
	}
	for _, slug := range want {
		if _, ok := set[slug]; !ok {
			return false
		}
	}
	return true
}

type configuredApp struct {
	registry    string
	publicBase  string
	routing     string
	root        string
	hubUpstream string
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
	if _, err := pool.Exec(ctx, `TRUNCATE deploys, auth_attempts, sessions, temporary_accounts, access_code_apps, access_codes, user_apps, users, public_apps`); err != nil {
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
	writeFile(t, filepath.Join(root, "apps", "hello", "dist", "index.html"), "hello")
	writeFile(t, filepath.Join(root, "apps", "hello", "dist", "assets", "app.js"), "js")
	upstream := cfg.hubUpstream
	if upstream == "" {
		stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("hub"))
		}))
		t.Cleanup(stub.Close)
		upstream = stub.URL
	}
	app, err := api.NewApp(api.Config{
		Pool:          pool,
		SiteRoot:      root,
		PublicBaseURL: publicBase,
		Routing:       cfg.routing,
		BcryptCost:    bcrypt.MinCost,
		CookieSecure:  "false",
		HubUpstream:   upstream,
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

func deleteReq(t *testing.T, app *fiber.App, path, cookieHeader string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	return do(t, app, req, cookieHeader, false)
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
