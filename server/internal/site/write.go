package site

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type appDoc struct {
	Slug        string `yaml:"slug"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Path        string `yaml:"path"`
	Icon        string `yaml:"icon"`
	Source      string `yaml:"source,omitempty"`
	Upstream    string `yaml:"upstream,omitempty"`
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
	current, err := s.Apps()
	if err != nil {
		return err
	}
	for _, existing := range current {
		if existing.Slug == app.Slug {
			return fmt.Errorf("slug %q is already registered", app.Slug)
		}
	}
	docs := make([]appDoc, 0, len(current)+1)
	for _, existing := range append(current, app) {
		docs = append(docs, appDoc(existing))
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(struct {
		Apps []appDoc `yaml:"apps"`
	}{Apps: docs}); err != nil {
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
	if _, err := s.Apps(); err != nil {
		if missing {
			_ = os.Remove(path)
		} else {
			_ = os.WriteFile(path, previous, 0o644)
		}
		return err
	}
	return nil
}
