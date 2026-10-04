package main

import (
	"context"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/jack-barr3tt/bouncer/db"
	"github.com/jack-barr3tt/bouncer/internal/api"
	"github.com/jack-barr3tt/bouncer/internal/deploy"
	"github.com/jack-barr3tt/bouncer/internal/envfile"
	"github.com/jack-barr3tt/bouncer/internal/store"
)

func main() {
	if err := envfile.Load(); err != nil {
		log.Fatal(err)
	}
	if len(os.Args) > 1 && os.Args[1] == "builder" {
		if err := deploy.ListenAndServeBuilder(); err != nil {
			log.Fatal(err)
		}
		return
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatal(err)
	}

	accounts, err := store.New(pool, 0)
	if err != nil {
		log.Fatal(err)
	}
	count, err := accounts.UserCount(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if count == 0 {
		username := strings.TrimSpace(os.Getenv("AUTH_BOOTSTRAP_USERNAME"))
		password := os.Getenv("AUTH_BOOTSTRAP_PASSWORD")
		if username == "" || len(password) < 8 {
			log.Fatal("no users exist: set AUTH_BOOTSTRAP_USERNAME and AUTH_BOOTSTRAP_PASSWORD (8 or more characters)")
		}
		if err := accounts.EnsureAdmin(ctx, username, password); err != nil {
			log.Fatal(err)
		}
		log.Printf("created admin %s", username)
	}

	root := os.Getenv("SITE_ROOT")
	if root == "" {
		root = defaultSiteRoot()
	}
	listen := os.Getenv("LISTEN_ADDR")
	if listen == "" {
		listen = ":8080"
	}
	base := strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL"))
	if base == "" {
		base = "http://127.0.0.1" + portSuffix(listen)
	}

	ships, err := deploy.FromEnv(pool, root, base)
	if err != nil {
		log.Fatal(err)
	}
	if err := ships.Recover(ctx); err != nil {
		log.Fatal(err)
	}
	if ships.Enabled() {
		log.Printf("git deploy enabled for branch %s", ships.Branch())
	}
	go ships.Poll(context.Background())

	app, err := api.NewApp(api.Config{
		Pool:           pool,
		SiteRoot:       root,
		HubDir:         os.Getenv("HUB_DIR"),
		PublicBaseURL:  base,
		TrustedProxies: splitList(os.Getenv("TRUSTED_PROXIES")),
		CookieSecure:   os.Getenv("COOKIE_SECURE"),
		Deploy:         ships,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("listening on %s", listen)
	if err := app.Listen(listen); err != nil {
		log.Fatal(err)
	}
}

func defaultSiteRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		if info, err := os.Stat(filepath.Join(dir, "apps", "hub")); err == nil && info.IsDir() {
			return dir
		}
		if info, err := os.Stat(filepath.Join(dir, "apps.yaml")); err == nil && !info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return cwd
		}
	}
}

func portSuffix(listen string) string {
	_, port, err := net.SplitHostPort(listen)
	if err != nil {
		port = strings.TrimPrefix(listen, ":")
	}
	if port == "" {
		return ""
	}
	return ":" + port
}

func splitList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
