package deploy

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jack-barr3tt/bouncer/internal/site"
)

func PublishDists(siteRoot, output string, targets []DistTarget) error {
	var staged []DistTarget
	for _, target := range targets {
		if err := placeDist(siteRoot, output, target); err != nil {
			_ = os.RemoveAll(filepath.Join(siteRoot, filepath.FromSlash(target.Dir), "dist.next"))
			for _, done := range staged {
				_ = os.RemoveAll(filepath.Join(siteRoot, filepath.FromSlash(done.Dir), "dist.next"))
			}
			return err
		}
		staged = append(staged, target)
	}
	for _, target := range staged {
		live := filepath.Join(siteRoot, filepath.FromSlash(target.Dir), "dist")
		if err := swapDir(live, live+".next"); err != nil {
			return err
		}
	}
	return nil
}

type DistTarget struct {
	Slug string
	Dir  string
}

func placeDist(siteRoot, output string, target DistTarget) error {
	dir, err := site.CleanSource(target.Dir)
	if err != nil {
		return err
	}
	if !safeSlug(target.Slug) {
		return fmt.Errorf("invalid slug %q", target.Slug)
	}
	src := filepath.Join(output, "apps", target.Slug, "dist")
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("dist for %s: %w", target.Slug, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("dist for %s is not a directory", target.Slug)
	}
	destDir := filepath.Join(siteRoot, filepath.FromSlash(dir))
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	staged := filepath.Join(destDir, "dist.next")
	if err := os.RemoveAll(staged); err != nil {
		return err
	}
	return copyTree(src, staged)
}

func Publish(siteRoot, checkout, output string, slugs []string) error {
	yamlPath := filepath.Join(checkout, "apps.yaml")
	if _, err := os.Stat(yamlPath); err != nil {
		return fmt.Errorf("apps.yaml: %w", err)
	}
	next, err := registrySlugs(checkout)
	if err != nil {
		return err
	}
	previous, err := registrySlugs(siteRoot)
	if err != nil {
		previous = nil
	}
	allowed := make(map[string]struct{}, len(next))
	for _, slug := range next {
		allowed[slug] = struct{}{}
	}
	var staged []string
	for _, slug := range slugs {
		if _, ok := allowed[slug]; !ok {
			continue
		}
		if err := stageDist(siteRoot, output, slug); err != nil {
			_ = os.RemoveAll(filepath.Join(siteRoot, "apps", slug, "dist.next"))
			for _, done := range staged {
				_ = os.RemoveAll(filepath.Join(siteRoot, "apps", done, "dist.next"))
			}
			return err
		}
		staged = append(staged, slug)
	}
	for _, slug := range staged {
		live := filepath.Join(siteRoot, "apps", slug, "dist")
		if err := swapDir(live, live+".next"); err != nil {
			return err
		}
	}
	if err := publishFile(yamlPath, filepath.Join(siteRoot, "apps.yaml")); err != nil {
		return err
	}
	return removeSlugs(siteRoot, missing(previous, next))
}

func registrySlugs(root string) ([]string, error) {
	path := filepath.Join(root, "apps.yaml")
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("apps.yaml is a directory")
	}
	opened, err := site.Open(root, "")
	if err != nil {
		return nil, err
	}
	apps, err := opened.Apps()
	if err != nil {
		return nil, err
	}
	slugs := make([]string, 0, len(apps))
	for _, app := range apps {
		slugs = append(slugs, app.Slug)
	}
	return slugs, nil
}

func missing(previous, next []string) []string {
	keep := make(map[string]struct{}, len(next))
	for _, slug := range next {
		keep[slug] = struct{}{}
	}
	var gone []string
	for _, slug := range previous {
		if _, ok := keep[slug]; !ok {
			gone = append(gone, slug)
		}
	}
	return gone
}

func stageDist(siteRoot, output, slug string) error {
	if !safeSlug(slug) {
		return fmt.Errorf("invalid slug %q", slug)
	}
	src := filepath.Join(output, "apps", slug, "dist")
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("dist for %s: %w", slug, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("dist for %s is not a directory", slug)
	}
	destDir := filepath.Join(siteRoot, "apps", slug)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	staged := filepath.Join(destDir, "dist.next")
	if err := os.RemoveAll(staged); err != nil {
		return err
	}
	return copyTree(src, staged)
}

func swapDir(live, staged string) error {
	prev := live + ".prev"
	if err := os.RemoveAll(prev); err != nil {
		return err
	}
	if _, err := os.Stat(live); err == nil {
		if err := os.Rename(live, prev); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(staged, live); err != nil {
		if _, statErr := os.Stat(prev); statErr == nil {
			_ = os.Rename(prev, live)
		}
		return err
	}
	return os.RemoveAll(prev)
}

func publishFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	body, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	next := dst + ".next"
	if err := os.WriteFile(next, body, 0o644); err != nil {
		return err
	}
	return os.Rename(next, dst)
}

func removeSlugs(siteRoot string, slugs []string) error {
	for _, slug := range slugs {
		if !safeSlug(slug) {
			return fmt.Errorf("invalid slug %q", slug)
		}
		if err := os.RemoveAll(filepath.Join(siteRoot, "apps", slug)); err != nil {
			return err
		}
	}
	return nil
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in dist: %s", rel)
		}
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
