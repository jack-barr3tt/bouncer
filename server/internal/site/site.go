package site

import (
	"os"
	"path/filepath"
	"strings"
)

type App struct {
	Slug        string `mapstructure:"slug" json:"slug"`
	Name        string `mapstructure:"name" json:"name"`
	Description string `mapstructure:"description" json:"description"`
	Path        string `mapstructure:"path" json:"path"`
	Icon        string `mapstructure:"icon" json:"icon"`
	Source      string `mapstructure:"source" json:"source,omitempty"`
	Upstream    string `mapstructure:"upstream" json:"upstream,omitempty"`
}

type Site struct {
	root string
	repo bool
}

func Open(root string) (*Site, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	repo := exists(filepath.Join(abs, "apps", "hub"))
	return &Site{root: abs, repo: repo}, nil
}

func (s *Site) Root() string { return s.root }

func (s *Site) Slugs() ([]string, error) {
	apps, err := s.Apps()
	if err != nil {
		return nil, err
	}
	slugs := make([]string, 0, len(apps))
	for _, app := range apps {
		slugs = append(slugs, app.Slug)
	}
	return slugs, nil
}

func (s *Site) IsApp(slug string) bool {
	ok, err := s.HasApp(slug)
	return err == nil && ok
}

func (s *Site) Known(slug string) bool {
	return s.IsApp(slug)
}

func (s *Site) AppDir(slug string) string {
	base := filepath.Join(s.root, "apps", slug)
	if apps, err := s.Apps(); err == nil {
		for _, app := range apps {
			if app.Slug == slug && app.Source != "" {
				base = filepath.Join(s.root, filepath.FromSlash(app.Source))
				break
			}
		}
	}
	dist := filepath.Join(base, "dist")
	if s.repo || exists(dist) {
		return dist
	}
	return base
}

func SafeJoin(dir, rel string) (string, bool) {
	if rel == "" {
		rel = "index.html"
	}
	if strings.Contains(rel, "\\") {
		return "", false
	}
	clean := filepath.Clean("/" + rel)
	if strings.HasPrefix(clean, "/..") || strings.Contains(clean, "..") {
		return "", false
	}
	rel = strings.TrimPrefix(clean, "/")
	for _, part := range strings.Split(rel, "/") {
		if part == "" || strings.HasPrefix(part, ".") {
			return "", false
		}
	}
	full := filepath.Join(dir, rel)
	root := filepath.Clean(dir)
	if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return "", false
	}
	return full, true
}

func exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
