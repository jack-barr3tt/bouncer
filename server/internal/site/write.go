package site

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type repoDoc struct {
	Name   string `yaml:"name"`
	Remote string `yaml:"remote"`
	Commit string `yaml:"commit,omitempty"`
	Branch string `yaml:"branch,omitempty"`
}

type appDoc struct {
	Slug        string `yaml:"slug"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Path        string `yaml:"path"`
	Icon        string `yaml:"icon"`
	Source      string `yaml:"source,omitempty"`
	Upstream    string `yaml:"upstream,omitempty"`
	Repo        string `yaml:"repo,omitempty"`
	Dir         string `yaml:"dir,omitempty"`
}

type registryDoc struct {
	Repos []repoDoc `yaml:"repos,omitempty"`
	Apps  []appDoc  `yaml:"apps"`
}

func (s *Site) AddApp(app App) error {
	if app.Source == "" {
		return fmt.Errorf("source is required")
	}
	clean, err := CleanSource(app.Source)
	if err != nil {
		return err
	}
	app.Source = clean
	app.Path = "/apps/" + app.Slug + "/"
	info, err := os.Stat(filepath.Join(s.root, filepath.FromSlash(app.Source), "package.json"))
	if err != nil || info.IsDir() {
		return fmt.Errorf("source has no package.json")
	}
	current, err := s.Load()
	if err != nil {
		return err
	}
	for _, existing := range current.Apps {
		if existing.Slug == app.Slug {
			return fmt.Errorf("slug %q is already registered", app.Slug)
		}
	}
	current.Apps = append(current.Apps, app)
	return s.writeRegistry(current)
}

func (s *Site) AddRepo(repo RepoRef, apps []App) error {
	repo.Name = strings.TrimSpace(repo.Name)
	repo.Remote = strings.TrimSpace(repo.Remote)
	repo.Commit = strings.TrimSpace(repo.Commit)
	repo.Branch = strings.TrimSpace(repo.Branch)
	if repo.Commit != "" && repo.Branch != "" {
		return fmt.Errorf("set a branch or a commit, not both")
	}
	if repo.Commit == "" && repo.Branch == "" {
		return fmt.Errorf("set a branch or a commit")
	}
	current, err := s.Load()
	if err != nil {
		return err
	}
	for _, existing := range current.Repos {
		if existing.Name == repo.Name {
			return fmt.Errorf("repository %q is already registered", repo.Name)
		}
	}
	seen := map[string]struct{}{}
	for _, existing := range current.Apps {
		seen[existing.Slug] = struct{}{}
	}
	for i := range apps {
		if _, dup := seen[apps[i].Slug]; dup {
			return fmt.Errorf("slug %q is already registered", apps[i].Slug)
		}
		seen[apps[i].Slug] = struct{}{}
		dir, err := CleanDir(apps[i].Dir)
		if err != nil {
			return err
		}
		apps[i].Dir = dir
		apps[i].Repo = repo.Name
		apps[i].Path = "/apps/" + apps[i].Slug + "/"
		apps[i].Source = ""
		apps[i].Upstream = ""
	}
	current.Repos = append(current.Repos, repo)
	current.Apps = append(current.Apps, apps...)
	return s.writeRegistry(current)
}

func (s *Site) writeRegistry(file Registry) error {
	doc := registryDoc{Apps: make([]appDoc, 0, len(file.Apps))}
	for _, repo := range file.Repos {
		doc.Repos = append(doc.Repos, repoDoc(repo))
	}
	for _, app := range file.Apps {
		doc.Apps = append(doc.Apps, appDoc(app))
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	path := filepath.Join(s.root, "apps.yaml")
	previous, readErr := os.ReadFile(path)
	missing := os.IsNotExist(readErr)
	if readErr != nil && !missing {
		return readErr
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return err
	}
	if _, err := s.Load(); err != nil {
		if missing {
			_ = os.Remove(path)
		} else {
			_ = os.WriteFile(path, previous, 0o644)
		}
		return err
	}
	return nil
}
