package stack

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type Field struct {
	Name     string `yaml:"name"`
	Value    string `yaml:"value"`
	Required bool   `yaml:"required"`
}

func ParseSpec(text string) ([]Field, error) {
	var fields []Field
	if err := yaml.Unmarshal([]byte(text), &fields); err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	for _, field := range fields {
		if field.Name == "" {
			return nil, fmt.Errorf("env spec: a field is missing a name")
		}
		if _, ok := seen[field.Name]; ok {
			return nil, fmt.Errorf("env spec: %s is listed twice", field.Name)
		}
		seen[field.Name] = struct{}{}
	}
	return fields, nil
}

// Render is the env file for a site to copy. A required field with no value is left blank.
func Render(fields []Field) string {
	var body string
	for _, field := range fields {
		body += field.Name + "=" + field.Value + "\n"
	}
	return body
}

func defaults(fields []Field) string {
	var body string
	for _, field := range fields {
		if field.Required {
			continue
		}
		body += field.Name + "=" + field.Value + "\n"
	}
	return body
}
