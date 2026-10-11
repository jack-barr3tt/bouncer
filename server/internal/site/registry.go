package site

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/jack-barr3tt/bouncer/schema"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var repoNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var commitPattern = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

var reservedSlugs = map[string]struct{}{
	"hub":    {},
	"assets": {},
	"code":   {},
}

var registrySchema = mustRegistrySchema()

func mustRegistrySchema() *jsonschema.Schema {
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(schema.Document))
	if err != nil {
		panic(err)
	}
	compiler := jsonschema.NewCompiler()
	url := "https://bouncer.local/apps.schema.json"
	if err := compiler.AddResource(url, doc); err != nil {
		panic(err)
	}
	compiled, err := compiler.Compile(url)
	if err != nil {
		panic(err)
	}
	return compiled
}

func (s *Site) Apps() ([]App, error) {
	file, err := s.Load()
	if err != nil {
		return nil, err
	}
	return file.Apps, nil
}

func (s *Site) Load() (Registry, error) {
	path := s.Root() + "/apps.yaml"
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Registry{}, nil
		}
		return Registry{}, err
	}

	var raw any
	if err := yaml.Unmarshal(body, &raw); err != nil {
		return Registry{}, fmt.Errorf("apps.yaml: %w", err)
	}
	if err := registrySchema.Validate(raw); err != nil {
		return Registry{}, fmt.Errorf("apps.yaml: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		var missing viper.ConfigFileNotFoundError
		if errors.As(err, &missing) || os.IsNotExist(err) {
			return Registry{}, nil
		}
		return Registry{}, fmt.Errorf("apps.yaml: %w", err)
	}
	var file Registry
	if err := v.Unmarshal(&file); err != nil {
		return Registry{}, fmt.Errorf("apps.yaml: %w", err)
	}

	names := make(map[string]struct{}, len(file.Repos))
	for i, repo := range file.Repos {
		if !repoNamePattern.MatchString(repo.Name) {
			return Registry{}, fmt.Errorf("repos[%d].name %q is not valid", i, repo.Name)
		}
		if _, dup := names[repo.Name]; dup {
			return Registry{}, fmt.Errorf("repos[%d].name %q is duplicated", i, repo.Name)
		}
		names[repo.Name] = struct{}{}
		if strings.TrimSpace(repo.Remote) == "" || strings.ContainsAny(repo.Remote, "\r\n\t ") || strings.HasPrefix(repo.Remote, "-") {
			return Registry{}, fmt.Errorf("repos[%d].remote is not valid", i)
		}
		switch {
		case repo.Commit != "" && repo.Branch != "":
			return Registry{}, fmt.Errorf("repos[%d] cannot set both commit and branch", i)
		case repo.Commit == "" && repo.Branch == "":
			return Registry{}, fmt.Errorf("repos[%d] must set commit or branch", i)
		case repo.Commit != "":
			if !commitPattern.MatchString(repo.Commit) {
				return Registry{}, fmt.Errorf("repos[%d].commit is not valid", i)
			}
		default:
			if !validRefBranch(repo.Branch) {
				return Registry{}, fmt.Errorf("repos[%d].branch is not valid", i)
			}
		}
	}

	seen := make(map[string]struct{}, len(file.Apps))
	for i, app := range file.Apps {
		if _, reserved := reservedSlugs[app.Slug]; reserved {
			return Registry{}, fmt.Errorf("apps[%d].slug %q is reserved", i, app.Slug)
		}
		if _, dup := seen[app.Slug]; dup {
			return Registry{}, fmt.Errorf("apps[%d].slug %q is duplicated", i, app.Slug)
		}
		seen[app.Slug] = struct{}{}
		want := "/apps/" + app.Slug + "/"
		if app.Path != want {
			return Registry{}, fmt.Errorf("apps[%d].path must be %q", i, want)
		}
		if app.Repo != "" && (app.Source != "" || app.Upstream != "") {
			return Registry{}, fmt.Errorf("apps[%d] cannot combine repo with source or upstream", i)
		}
		if app.Source != "" && app.Upstream != "" {
			return Registry{}, fmt.Errorf("apps[%d] cannot set both source and upstream", i)
		}
		if app.Dir != "" && app.Repo == "" {
			return Registry{}, fmt.Errorf("apps[%d].dir requires repo", i)
		}
		if app.Upstream != "" {
			clean, err := CleanUpstream(app.Upstream)
			if err != nil {
				return Registry{}, fmt.Errorf("apps[%d].upstream: %w", i, err)
			}
			file.Apps[i].Upstream = clean
		}
		if app.Repo != "" {
			if _, ok := names[app.Repo]; !ok {
				return Registry{}, fmt.Errorf("apps[%d].repo %q is not listed", i, app.Repo)
			}
			clean, err := CleanDir(app.Dir)
			if err != nil {
				return Registry{}, fmt.Errorf("apps[%d].dir: %w", i, err)
			}
			file.Apps[i].Dir = clean
		}
		if app.Source == "" {
			continue
		}
		clean, err := CleanSource(app.Source)
		if err != nil {
			return Registry{}, fmt.Errorf("apps[%d].source: %w", i, err)
		}
		file.Apps[i].Source = clean
	}
	return file, nil
}

func CleanUpstream(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Opaque != "" {
		return "", fmt.Errorf("%q must be an absolute http or https origin", raw)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("%q must be an absolute http or https origin", raw)
	}
	if parsed.User != nil {
		return "", fmt.Errorf("%q must not include a username or password", raw)
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", fmt.Errorf("%q must not include a path", raw)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("%q must not include a query or fragment", raw)
	}
	parsed.Path = ""
	parsed.RawPath = ""
	return parsed.String(), nil
}

func CleanDir(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" || dir == "." {
		return "", nil
	}
	if strings.Contains(dir, "\\") || strings.HasPrefix(dir, "/") {
		return "", fmt.Errorf("%q must be a directory inside the repository", dir)
	}
	clean := path.Clean(dir)
	for _, part := range strings.Split(clean, "/") {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("%q must be a directory inside the repository", dir)
		}
	}
	return clean, nil
}

func validRefBranch(branch string) bool {
	if branch == "" || strings.Contains(branch, "..") || strings.HasPrefix(branch, "/") || strings.HasSuffix(branch, "/") {
		return false
	}
	for _, r := range branch {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.' || r == '_' || r == '/' || r == '-':
		default:
			return false
		}
	}
	return true
}

func CleanSource(source string) (string, error) {
	source = strings.TrimSpace(source)
	if source == "" || strings.Contains(source, "\\") || strings.HasPrefix(source, "/") {
		return "", fmt.Errorf("%q must be a directory inside apps/", source)
	}
	clean := path.Clean(source)
	if !strings.HasPrefix(clean, "apps/") {
		return "", fmt.Errorf("%q must be a directory inside apps/", source)
	}
	rest := strings.TrimPrefix(clean, "apps/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("%q must be a directory inside apps/", source)
	}
	if _, reserved := reservedSlugs[parts[0]]; reserved {
		return "", fmt.Errorf("%q uses a reserved directory", source)
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("%q must be a directory inside apps/", source)
		}
	}
	return clean, nil
}

func (s *Site) Lookup(slug string) (App, bool, error) {
	apps, err := s.Apps()
	if err != nil {
		return App{}, false, err
	}
	for _, app := range apps {
		if app.Slug == slug {
			return app, true, nil
		}
	}
	return App{}, false, nil
}

func (s *Site) HasApp(slug string) (bool, error) {
	apps, err := s.Apps()
	if err != nil {
		return false, err
	}
	for _, app := range apps {
		if app.Slug == slug {
			return true, nil
		}
	}
	return false, nil
}
