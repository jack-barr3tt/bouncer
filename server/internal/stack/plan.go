package stack

import (
	"encoding/json"
	"os"
	"strings"
)

type choice struct {
	apply         bool
	env           string
	noticeVersion string
	noticeKeys    []string
	replace       bool
}

type Notice struct {
	Version string   `json:"version"`
	Keys    []string `json:"keys"`
}

// MissingRequired returns required fields that are absent or blank in envText.
func MissingRequired(envText string, fields []Field) []string {
	have := envValues(envText)
	var missing []string
	for _, field := range fields {
		if !field.Required {
			continue
		}
		if strings.TrimSpace(have[field.Name]) == "" {
			missing = append(missing, field.Name)
		}
	}
	return missing
}

func choose(pin, running, envText string, embedded []Field, candidate string, candidateFields []Field) choice {
	current := pin
	if !semver(current) {
		current = ""
	}
	if current == "" && semver(running) {
		current = running
	}

	noticeVersion := ""
	var noticeKeys []string
	desired := current
	example := embedded
	if semver(candidate) && (desired == "" || newer(candidate, desired)) {
		keys := MissingRequired(envText, candidateFields)
		if len(keys) > 0 {
			noticeVersion = candidate
			noticeKeys = keys
		} else {
			desired = candidate
			example = candidateFields
		}
	}

	if desired == "" {
		body, _ := MergeEnv(envText, defaults(embedded))
		return choice{apply: true, env: body}
	}

	if keys := MissingRequired(envText, example); len(keys) > 0 {
		if noticeVersion == "" {
			noticeVersion = desired
			noticeKeys = keys
		}
		next := envText
		if !semver(pin) && semver(running) {
			next = SetKey(next, "IMAGE_TAG", running)
		}
		return choice{env: next, noticeVersion: noticeVersion, noticeKeys: noticeKeys}
	}

	body, _ := MergeEnv(envText, defaults(example))
	body = SetKey(body, "IMAGE_TAG", desired)
	return choice{
		apply:         true,
		env:           body,
		noticeVersion: noticeVersion,
		noticeKeys:    noticeKeys,
		replace:       desired != pin,
	}
}

func SetKey(body, key, value string) string {
	lines := strings.Split(body, "\n")
	found := false
	for i, line := range lines {
		if envKey(line) != key {
			continue
		}
		lines[i] = key + "=" + value
		found = true
	}
	next := strings.Join(lines, "\n")
	if found {
		return next
	}
	if next != "" && !strings.HasSuffix(next, "\n") {
		next += "\n"
	}
	return next + key + "=" + value + "\n"
}

// AutoUpdate reports whether the reconciler should move to a newer published version.
// A missing value stays on, so an existing site keeps updating until an admin turns it off.
func AutoUpdate(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "false", "0", "no", "off":
		return false
	default:
		return true
	}
}

// EnvValue returns the value of one key in an env file. A missing key is empty.
func EnvValue(text, key string) string {
	return envValues(text)[key]
}

func envValues(text string) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		key := envKey(line)
		if key == "" {
			continue
		}
		_, rest, _ := strings.Cut(line, "=")
		rest = strings.TrimSpace(rest)
		if before, _, ok := strings.Cut(rest, "#"); ok {
			rest = strings.TrimSpace(before)
		}
		values[key] = rest
	}
	return values
}

func semver(value string) bool {
	_, ok := parseSemver(value)
	return ok
}

func newer(a, b string) bool {
	left, ok := parseSemver(a)
	if !ok {
		return false
	}
	right, ok := parseSemver(b)
	if !ok {
		return true
	}
	for i := 0; i < 3; i++ {
		if left[i] == right[i] {
			continue
		}
		return left[i] > right[i]
	}
	return false
}

func parseSemver(value string) ([3]int, bool) {
	var parsed [3]int
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return parsed, false
	}
	for i, part := range parts {
		if part == "" || strings.TrimLeft(part, "0123456789") != "" {
			return parsed, false
		}
		n := 0
		for _, ch := range part {
			n = n*10 + int(ch-'0')
		}
		parsed[i] = n
	}
	return parsed, true
}

func ReadNotice(path string) (Notice, bool, error) {
	body, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Notice{}, false, nil
	}
	if err != nil {
		return Notice{}, false, err
	}
	var notice Notice
	if err := json.Unmarshal(body, &notice); err != nil {
		return Notice{}, false, err
	}
	if notice.Version == "" || len(notice.Keys) == 0 {
		return Notice{}, false, nil
	}
	return notice, true, nil
}

func writeNotice(path string, notice Notice) error {
	if notice.Version == "" || len(notice.Keys) == 0 {
		err := os.Remove(path)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	body, err := json.Marshal(notice)
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(path, body, 0o644)
}
