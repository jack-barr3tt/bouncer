package deploy

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var reservedSlugs = map[string]struct{}{
	"hub":    {},
	"assets": {},
	"code":   {},
}

var diffSlug = regexp.MustCompile(`^apps/([^/]+)/`)

func ShouldBuildAll(before string) bool {
	if before == "" {
		return true
	}
	for _, r := range before {
		if r != '0' {
			return false
		}
	}
	return true
}

func ListAppSlugs(appsDir string) ([]string, error) {
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var slugs []string
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || strings.HasPrefix(name, ".") {
			continue
		}
		if _, reserved := reservedSlugs[name]; reserved {
			continue
		}
		if _, err := os.Stat(filepath.Join(appsDir, name, "package.json")); err != nil {
			continue
		}
		slugs = append(slugs, name)
	}
	sort.Strings(slugs)
	return slugs, nil
}

func SlugsFromDiff(diff string, apps []string) []string {
	known := make(map[string]struct{}, len(apps))
	for _, app := range apps {
		known[app] = struct{}{}
	}
	found := map[string]struct{}{}
	for _, line := range strings.Split(diff, "\n") {
		match := diffSlug.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		if _, ok := known[match[1]]; ok {
			found[match[1]] = struct{}{}
		}
	}
	slugs := make([]string, 0, len(found))
	for slug := range found {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	return slugs
}

func safeSlug(slug string) bool {
	if slug == "" || strings.HasPrefix(slug, ".") {
		return false
	}
	if _, reserved := reservedSlugs[slug]; reserved {
		return false
	}
	for _, r := range slug {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
		default:
			return false
		}
	}
	return true
}
