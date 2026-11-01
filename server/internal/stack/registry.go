package stack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Catalog interface {
	Latest(ctx context.Context) (string, error)
	Example(ctx context.Context, version string) (string, error)
}

func liveCatalog(run Runner) Catalog {
	registry := os.Getenv("REGISTRY")
	if registry == "" {
		registry = "ghcr.io/jack-barr3tt"
	}
	return &registryCatalog{
		registry: registry,
		client:   http.DefaultClient,
		run:      run,
	}
}

type registryCatalog struct {
	registry string
	client   *http.Client
	run      Runner
}

func (c *registryCatalog) Latest(ctx context.Context) (string, error) {
	host, repo, err := splitRegistry(c.registry)
	if err != nil {
		return "", err
	}
	return latestVersion(ctx, c.client, host, repo+"/bouncer")
}

func (c *registryCatalog) Example(ctx context.Context, version string) (string, error) {
	image := strings.TrimRight(c.registry, "/") + "/bouncer:" + version
	if err := c.run(ctx, "docker", "pull", image); err != nil {
		return "", err
	}
	name := "bouncer-env-" + strings.ReplaceAll(version, ".", "-")
	_ = c.run(context.Background(), "docker", "rm", "-f", name)
	if err := c.run(ctx, "docker", "create", "--name", name, image); err != nil {
		return "", err
	}
	defer func() { _ = c.run(context.Background(), "docker", "rm", "-f", name) }()
	dest := filepath.Join(os.TempDir(), name+".env")
	defer os.Remove(dest)
	if err := c.run(ctx, "docker", "cp", name+":/opt/bouncer/env.yaml", dest); err != nil {
		return "", err
	}
	body, err := os.ReadFile(dest)
	return string(body), err
}

func splitRegistry(registry string) (string, string, error) {
	registry = strings.TrimPrefix(strings.TrimPrefix(registry, "https://"), "http://")
	registry = strings.Trim(registry, "/")
	host, repo, ok := strings.Cut(registry, "/")
	if !ok || host == "" || repo == "" {
		return "", "", fmt.Errorf("registry %q has no repository", registry)
	}
	return host, repo, nil
}

func latestVersion(ctx context.Context, client *http.Client, host, repository string) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	tokenURL := "https://" + host + "/token?" + url.Values{
		"service": []string{host},
		"scope":   []string{"repository:" + repository + ":pull"},
	}.Encode()
	token, err := getJSON(ctx, client, tokenURL, nil)
	if err != nil {
		return "", err
	}
	var auth struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(token, &auth); err != nil {
		return "", err
	}
	if auth.Token == "" {
		return "", fmt.Errorf("registry token is empty")
	}
	tagsURL := "https://" + host + "/v2/" + repository + "/tags/list?n=1000"
	body, err := getJSON(ctx, client, tagsURL, http.Header{"Authorization": []string{"Bearer " + auth.Token}})
	if err != nil {
		return "", err
	}
	var listed struct {
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal(body, &listed); err != nil {
		return "", err
	}
	best := ""
	for _, tag := range listed.Tags {
		if !semver(tag) {
			continue
		}
		if best == "" || newer(tag, best) {
			best = tag
		}
	}
	if best == "" {
		return "", fmt.Errorf("no published version")
	}
	return best, nil
}

func getJSON(ctx context.Context, client *http.Client, raw string, header http.Header) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	for key, values := range header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry %s: %s", res.Status, strings.TrimSpace(string(body)))
	}
	return body, nil
}
