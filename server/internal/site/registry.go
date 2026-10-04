package site

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/jack-barr3tt/bouncer/schema"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

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
	path := s.Root() + "/apps.yaml"
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var raw any
	if err := yaml.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("apps.yaml: %w", err)
	}
	if err := registrySchema.Validate(raw); err != nil {
		return nil, fmt.Errorf("apps.yaml: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		var missing viper.ConfigFileNotFoundError
		if errors.As(err, &missing) || os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("apps.yaml: %w", err)
	}
	var file struct {
		Apps []App `mapstructure:"apps"`
	}
	if err := v.Unmarshal(&file); err != nil {
		return nil, fmt.Errorf("apps.yaml: %w", err)
	}

	seen := make(map[string]struct{}, len(file.Apps))
	for i, app := range file.Apps {
		if _, reserved := reservedSlugs[app.Slug]; reserved {
			return nil, fmt.Errorf("apps[%d].slug %q is reserved", i, app.Slug)
		}
		if _, dup := seen[app.Slug]; dup {
			return nil, fmt.Errorf("apps[%d].slug %q is duplicated", i, app.Slug)
		}
		seen[app.Slug] = struct{}{}
		want := "/apps/" + app.Slug + "/"
		if app.Path != want {
			return nil, fmt.Errorf("apps[%d].path must be %q", i, want)
		}
		if app.Source != "" && app.Upstream != "" {
			return nil, fmt.Errorf("apps[%d] cannot set both source and upstream", i)
		}
		if app.Upstream != "" {
			clean, err := CleanUpstream(app.Upstream)
			if err != nil {
				return nil, fmt.Errorf("apps[%d].upstream: %w", i, err)
			}
			file.Apps[i].Upstream = clean
		}
		if app.Source == "" {
			continue
		}
		clean, err := CleanSource(app.Source)
		if err != nil {
			return nil, fmt.Errorf("apps[%d].source: %w", i, err)
		}
		file.Apps[i].Source = clean
	}
	return file.Apps, nil
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
