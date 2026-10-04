package envfile

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Load reads .env into the environment. It looks in the working directory, then
// in each parent, and stops at the repository root. A variable that is already
// set is left as it is. A missing file is not an error. A relative SITE_ROOT
// is resolved from the directory that held .env.
func Load() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	envDir := ""
	for {
		err := godotenv.Load(filepath.Join(dir, ".env"))
		if err == nil {
			envDir = dir
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if isRepoRoot(dir) {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return resolveSiteRoot(envDir)
}

func resolveSiteRoot(envDir string) error {
	root := os.Getenv("SITE_ROOT")
	if root == "" || filepath.IsAbs(root) || envDir == "" {
		return nil
	}
	abs, err := filepath.Abs(filepath.Join(envDir, root))
	if err != nil {
		return err
	}
	return os.Setenv("SITE_ROOT", abs)
}

func isRepoRoot(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && info.IsDir()
}
