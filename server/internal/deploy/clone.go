package deploy

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Clone struct {
	Path   string
	Remote string
	Branch string
	Apps   []string
}

func (s *Service) AddClone(ctx context.Context, remote, name, branch string) (Clone, error) {
	remote = strings.TrimSpace(remote)
	name = strings.TrimSpace(name)
	branch = strings.TrimSpace(branch)
	if branch == "" {
		branch = "main"
	}
	if !validRemote(remote) {
		return Clone{}, fmt.Errorf("remote is not valid")
	}
	if !safeDirName(name) {
		return Clone{}, fmt.Errorf("folder name is not valid")
	}
	if !validBranch(branch) {
		return Clone{}, fmt.Errorf("branch is not valid")
	}
	if strings.TrimSpace(s.cfg.SiteRoot) == "" {
		return Clone{}, fmt.Errorf("site directory is not set")
	}
	appsDir := filepath.Join(s.cfg.SiteRoot, "apps")
	dest := filepath.Join(appsDir, name)
	if _, err := os.Stat(dest); err == nil {
		return Clone{}, fmt.Errorf("folder already exists")
	} else if !os.IsNotExist(err) {
		return Clone{}, err
	}
	if err := os.MkdirAll(appsDir, 0o755); err != nil {
		return Clone{}, err
	}
	cloner := Repo{Remote: remote, KeyFile: s.cfg.KeyFile, RuntimeDir: s.cfg.RuntimeDir, Dir: appsDir}
	if err := cloner.prepareSSH(); err != nil {
		return Clone{}, err
	}
	if _, err := cloner.git(ctx, appsDir, "clone", "--branch", branch, "--single-branch", remote, name); err != nil {
		_ = os.RemoveAll(dest)
		return Clone{}, fmt.Errorf("could not clone that repository")
	}
	apps, err := detectAppSources(dest, "apps/"+name)
	if err != nil {
		return Clone{}, err
	}
	return Clone{Path: "apps/" + name, Remote: remote, Branch: branch, Apps: apps}, nil
}

func safeDirName(name string) bool {
	if !safeSlug(name) {
		return false
	}
	if _, reserved := reservedSlugs[name]; reserved {
		return false
	}
	return true
}

func detectAppSources(cloneRoot, siteRel string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(cloneRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != cloneRoot {
				switch entry.Name() {
				case ".git", "node_modules", "dist":
					return filepath.SkipDir
				}
			}
			return nil
		}
		if entry.Name() != "package.json" {
			return nil
		}
		rel, err := filepath.Rel(cloneRoot, filepath.Dir(path))
		if err != nil {
			return err
		}
		if rel == "." {
			found = append(found, siteRel)
			return nil
		}
		found = append(found, siteRel+"/"+filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(found)
	return found, err
}
