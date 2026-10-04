package deploy

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Repo struct {
	Dir        string
	Remote     string
	Branch     string
	KeyFile    string
	RuntimeDir string
	script     string
}

func (r *Repo) Sync(ctx context.Context, sha string) (string, error) {
	if err := os.MkdirAll(r.Dir, 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(r.hooks(), 0o755); err != nil {
		return "", err
	}
	if err := r.prepareSSH(); err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(r.Dir, ".git")); err != nil {
		if _, err := r.git(ctx, r.Dir, "init"); err != nil {
			return "", err
		}
	}
	if _, err := r.git(ctx, r.Dir, "remote", "get-url", "origin"); err != nil {
		if _, err := r.git(ctx, r.Dir, "remote", "add", "origin", r.Remote); err != nil {
			return "", err
		}
	} else if _, err := r.git(ctx, r.Dir, "remote", "set-url", "origin", r.Remote); err != nil {
		return "", err
	}
	ref := "refs/heads/" + r.Branch
	remoteRef := "refs/remotes/origin/" + r.Branch
	spec := "+" + ref + ":" + remoteRef
	if _, err := r.git(ctx, r.Dir, "fetch", "--prune", "origin", spec); err != nil {
		return "", err
	}
	tip, err := r.rev(ctx, remoteRef)
	if err != nil {
		return "", err
	}
	resolved := tip
	if sha != "" {
		if !validSHA(sha) {
			return "", errInvalidSHA
		}
		if _, err := r.git(ctx, r.Dir, "cat-file", "-e", sha+"^{commit}"); err != nil {
			if _, ferr := r.git(ctx, r.Dir, "fetch", "origin", sha); ferr != nil {
				return "", fmt.Errorf("fetch %s: %w", sha, ferr)
			}
		}
		if _, err := r.git(ctx, r.Dir, "merge-base", "--is-ancestor", sha, remoteRef); err != nil {
			return "", fmt.Errorf("commit is not on %s", r.Branch)
		}
		resolved = sha
	}
	if _, err := r.git(ctx, r.Dir, "checkout", "--detach", "--force", resolved); err != nil {
		return "", err
	}
	if _, err := r.git(ctx, r.Dir, "clean", "-fdx"); err != nil {
		return "", err
	}
	return resolved, nil
}

func (r *Repo) Tip(ctx context.Context) (string, error) {
	if err := r.prepareSSH(); err != nil {
		return "", err
	}
	dir := r.RuntimeDir
	if dir == "" {
		dir = os.TempDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	out, err := r.git(ctx, dir, "ls-remote", r.Remote, "refs/heads/"+r.Branch)
	if err != nil {
		return "", err
	}
	fields := strings.Fields(out)
	if len(fields) == 0 || !validSHA(fields[0]) {
		return "", fmt.Errorf("branch %s was not found", r.Branch)
	}
	return fields[0], nil
}

func (r *Repo) DiffNames(ctx context.Context, from, to string) (string, error) {
	if !validSHA(from) || !validSHA(to) {
		return "", errInvalidSHA
	}
	return r.git(ctx, r.Dir, "diff", "--name-only", from, to)
}

func (r *Repo) rev(ctx context.Context, ref string) (string, error) {
	out, err := r.git(ctx, r.Dir, "rev-parse", ref)
	if err != nil {
		return "", err
	}
	sha := strings.TrimSpace(out)
	if !validSHA(sha) {
		return "", fmt.Errorf("unexpected revision %q", sha)
	}
	return sha, nil
}

func (r *Repo) hooks() string {
	return filepath.Join(r.RuntimeDir, "hooks")
}

func (r *Repo) prepareSSH() error {
	info, err := os.Stat(r.KeyFile)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		r.script = ""
		if sshRemote(r.Remote) {
			return fmt.Errorf("deploy key is missing")
		}
		return nil
	}
	if err := os.MkdirAll(r.RuntimeDir, 0o700); err != nil {
		return err
	}
	body, err := os.ReadFile(r.KeyFile)
	if err != nil {
		return err
	}
	keyPath := filepath.Join(r.RuntimeDir, "id")
	if strings.Contains(keyPath, "'") {
		return fmt.Errorf("key path is not valid")
	}
	if err := os.WriteFile(keyPath, body, 0o600); err != nil {
		return err
	}
	script := "#!/bin/sh\nexec ssh -i '" + keyPath + "' -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new \"$@\"\n"
	path := filepath.Join(r.RuntimeDir, "ssh")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		return err
	}
	r.script = path
	return nil
}

func sshRemote(remote string) bool {
	return strings.HasPrefix(remote, "git@") || strings.HasPrefix(remote, "ssh://")
}

func (r *Repo) git(ctx context.Context, dir string, args ...string) (string, error) {
	if err := os.MkdirAll(r.hooks(), 0o755); err != nil {
		return "", err
	}
	base := []string{"-c", "safe.directory=*", "-c", "protocol.file.allow=always", "-c", "core.hooksPath=" + r.hooks()}
	cmd := exec.CommandContext(ctx, "git", append(base, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if r.script != "" {
		cmd.Env = append(cmd.Env, "GIT_SSH="+r.script)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w\n%s%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String(), nil
}
