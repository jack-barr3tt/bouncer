package deploy

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var repoName = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type DetectedApp struct {
	Dir string
}

type Preview struct {
	Name   string
	Remote string
	Commit string
	Branch string
	Apps   []DetectedApp
}

type PreviewOptions struct {
	Remote     string
	Name       string
	Commit     string
	Branch     string
	KeyFile    string
	RuntimeDir string
	Ambient    bool
}

func (s *Service) PreviewRepo(ctx context.Context, remote, name, commit, branch string) (Preview, error) {
	return FetchRepo(ctx, PreviewOptions{
		Remote:     remote,
		Name:       name,
		Commit:     commit,
		Branch:     branch,
		KeyFile:    s.cfg.KeyFile,
		RuntimeDir: s.cfg.RuntimeDir,
	})
}

func FetchRepo(ctx context.Context, opts PreviewOptions) (Preview, error) {
	opts.Remote = strings.TrimSpace(opts.Remote)
	opts.Name = strings.TrimSpace(opts.Name)
	opts.Commit = strings.TrimSpace(opts.Commit)
	opts.Branch = strings.TrimSpace(opts.Branch)
	if !validRemote(opts.Remote) {
		return Preview{}, fmt.Errorf("remote is not valid")
	}
	if !validRepoName(opts.Name) {
		return Preview{}, fmt.Errorf("name is not valid")
	}
	if opts.Commit != "" && opts.Branch != "" {
		return Preview{}, fmt.Errorf("set a branch or a commit, not both")
	}
	if opts.Commit == "" && opts.Branch == "" {
		return Preview{}, fmt.Errorf("set a branch or a commit")
	}
	if opts.Commit != "" && (len(opts.Commit) != 40 || !validSHA(opts.Commit)) {
		return Preview{}, fmt.Errorf("commit is not valid")
	}
	if opts.Branch != "" && !validBranch(opts.Branch) {
		return Preview{}, fmt.Errorf("branch is not valid")
	}
	dir, err := os.MkdirTemp("", "bouncer-repo-")
	if err != nil {
		return Preview{}, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	runtime := opts.RuntimeDir
	if runtime == "" {
		runtime = os.TempDir()
	}
	repo := Repo{
		Dir:        dir,
		Remote:     opts.Remote,
		Branch:     opts.Branch,
		KeyFile:    opts.KeyFile,
		RuntimeDir: runtime,
		Ambient:    opts.Ambient,
	}
	if _, err := repo.Sync(ctx, opts.Commit); err != nil {
		return Preview{}, fmt.Errorf("could not fetch that repository: %w", err)
	}
	dirs, err := detectAppDirs(dir)
	if err != nil {
		return Preview{}, err
	}
	apps := make([]DetectedApp, 0, len(dirs))
	for _, item := range dirs {
		apps = append(apps, DetectedApp{Dir: item})
	}
	return Preview{
		Name:   opts.Name,
		Remote: opts.Remote,
		Commit: opts.Commit,
		Branch: opts.Branch,
		Apps:   apps,
	}, nil
}

func validRepoName(name string) bool {
	return repoName.MatchString(name)
}

func detectAppDirs(root string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(root, func(candidate string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if candidate != root {
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
		rel, err := filepath.Rel(root, filepath.Dir(candidate))
		if err != nil {
			return err
		}
		if rel == "." {
			found = append(found, ".")
			return nil
		}
		found = append(found, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(found)
	return found, err
}
