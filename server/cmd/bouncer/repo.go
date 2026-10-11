package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jack-barr3tt/bouncer/internal/deploy"
	"github.com/jack-barr3tt/bouncer/internal/site"
)

func runRepo(args []string, in io.Reader, out io.Writer) error {
	if len(args) == 0 || args[0] != "add" {
		return fmt.Errorf("usage: bouncer repo add <remote> [--site DIR] [--name NAME] (--branch BRANCH | --commit SHA) [--app DIR=SLUG]")
	}
	remote, opts, err := parseRepoAdd(args[1:])
	if err != nil {
		return err
	}
	interactive := isTerminal(in)
	reader := bufio.NewReader(in)
	if opts.site == "" {
		opts.site = os.Getenv("SITE_ROOT")
	}
	if opts.site == "" {
		opts.site = "."
	}
	if remote == "" {
		if !interactive {
			return fmt.Errorf("remote is required")
		}
		remote = ask(reader, out, "Git remote", "")
	}
	if opts.name == "" {
		opts.name = repoNameFromRemote(remote)
		if interactive {
			opts.name = ask(reader, out, "Repository name", opts.name)
		}
	}
	if opts.commit == "" && opts.branch == "" {
		if !interactive {
			return fmt.Errorf("set --branch or --commit")
		}
		choice := ask(reader, out, "Track a branch, or pin a commit? [branch/commit]", "branch")
		if choice == "commit" {
			opts.commit = ask(reader, out, "Commit", "")
		} else {
			opts.branch = ask(reader, out, "Branch", "main")
		}
	}
	key, ambient := repoKey()
	preview, err := deploy.FetchRepo(context.Background(), deploy.PreviewOptions{
		Remote:     remote,
		Name:       opts.name,
		Commit:     opts.commit,
		Branch:     opts.branch,
		KeyFile:    key,
		RuntimeDir: os.TempDir(),
		Ambient:    ambient,
	})
	if err != nil {
		return err
	}
	selected, err := chooseApps(preview, opts.apps, interactive, reader, out)
	if err != nil {
		return err
	}
	opened, err := site.Open(opts.site)
	if err != nil {
		return err
	}
	if err := opened.AddRepo(site.RepoRef{
		Name:   preview.Name,
		Remote: preview.Remote,
		Commit: preview.Commit,
		Branch: preview.Branch,
	}, selected); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "Added %s to %s\n", preview.Name, filepath.Join(opened.Root(), "apps.yaml"))
	if _, err := os.Stat(filepath.Join(opened.Root(), ".git")); err == nil {
		_, _ = fmt.Fprintln(out, "Commit apps.yaml to publish this reference.")
	}
	return nil
}

type repoAddOptions struct {
	site   string
	name   string
	commit string
	branch string
	apps   []string
}

func parseRepoAdd(args []string) (string, repoAddOptions, error) {
	var opts repoAddOptions
	var remote string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--site" || arg == "--name" || arg == "--commit" || arg == "--branch" || arg == "--app":
			if i+1 >= len(args) {
				return "", opts, fmt.Errorf("%s needs a value", arg)
			}
			i++
			value := args[i]
			switch arg {
			case "--site":
				opts.site = value
			case "--name":
				opts.name = value
			case "--commit":
				opts.commit = value
			case "--branch":
				opts.branch = value
			case "--app":
				opts.apps = append(opts.apps, value)
			}
		case strings.HasPrefix(arg, "--"):
			return "", opts, fmt.Errorf("unknown flag %s", arg)
		default:
			if remote != "" {
				return "", opts, fmt.Errorf("unexpected argument %s", arg)
			}
			remote = arg
		}
	}
	return remote, opts, nil
}

func chooseApps(preview deploy.Preview, specs []string, interactive bool, in *bufio.Reader, out io.Writer) ([]site.App, error) {
	if len(specs) > 0 {
		apps := make([]site.App, 0, len(specs))
		for _, spec := range specs {
			dir, slug, ok := strings.Cut(spec, "=")
			if !ok || slug == "" {
				return nil, fmt.Errorf("--app must be DIR=SLUG")
			}
			if dir == "" {
				dir = "."
			}
			if !containsDir(preview.Apps, dir) {
				return nil, fmt.Errorf("no app at %s", dir)
			}
			apps = append(apps, defaultApp(dir, slug))
		}
		return apps, nil
	}
	chosen := preview.Apps
	if interactive {
		_, _ = fmt.Fprintln(out, "Apps:")
		for _, app := range preview.Apps {
			_, _ = fmt.Fprintf(out, "  %s\n", app.Dir)
		}
		answer := ask(in, out, "Register all of these apps? [yes/no]", "yes")
		if answer == "no" || answer == "n" {
			chosen = nil
			for _, app := range preview.Apps {
				keep := ask(in, out, "Register "+app.Dir+"? [yes/no]", "yes")
				if keep == "yes" || keep == "y" || keep == "" {
					chosen = append(chosen, app)
				}
			}
		}
	}
	apps := make([]site.App, 0, len(chosen))
	for _, app := range chosen {
		slug := slugFromDir(app.Dir, preview.Name)
		apps = append(apps, defaultApp(app.Dir, slug))
	}
	return apps, nil
}

func containsDir(apps []deploy.DetectedApp, dir string) bool {
	for _, app := range apps {
		if app.Dir == dir {
			return true
		}
	}
	return false
}

func defaultApp(dir, slug string) site.App {
	name := displayName(slug)
	icon := "A"
	if name != "" {
		icon = string([]rune(name)[0])
	}
	app := site.App{
		Slug:        slug,
		Name:        name,
		Description: name,
		Icon:        icon,
		Dir:         dir,
	}
	if dir == "." {
		app.Dir = ""
	}
	return app
}

func slugFromDir(dir, repoName string) string {
	base := dir
	if base == "" || base == "." {
		base = repoName
	} else {
		base = path.Base(base)
	}
	return strings.ToLower(base)
}

func displayName(slug string) string {
	parts := strings.Split(slug, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func repoNameFromRemote(remote string) string {
	remote = strings.TrimSuffix(strings.TrimSpace(remote), "/")
	remote = strings.TrimSuffix(remote, ".git")
	if i := strings.LastIndex(remote, ":"); i >= 0 && strings.Contains(remote[:i], "@") {
		remote = remote[i+1:]
	}
	if i := strings.LastIndex(remote, "/"); i >= 0 {
		remote = remote[i+1:]
	}
	return strings.ToLower(remote)
}

func repoKey() (string, bool) {
	key := os.Getenv("GIT_SSH_KEY_FILE")
	if key == "" {
		key = "/run/bouncer/git-key"
	}
	info, err := os.Stat(key)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return "", true
	}
	return key, false
}

func ask(in *bufio.Reader, out io.Writer, label, fallback string) string {
	if fallback == "" {
		_, _ = fmt.Fprintf(out, "%s: ", label)
	} else {
		_, _ = fmt.Fprintf(out, "%s [%s]: ", label, fallback)
	}
	line, _ := in.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return fallback
	}
	return line
}

func isTerminal(in io.Reader) bool {
	file, ok := in.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
