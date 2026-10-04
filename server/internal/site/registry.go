package site

import (
	"errors"
	"fmt"
	"os"
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
	}
	return file.Apps, nil
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
