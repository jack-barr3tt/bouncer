package deploy

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

func verifySignature(secret string, body []byte, header string) bool {
	if secret == "" || !strings.HasPrefix(header, "sha256=") {
		return false
	}
	mac, err := hex.DecodeString(strings.TrimPrefix(header, "sha256="))
	if err != nil {
		return false
	}
	sum := hmac.New(sha256.New, []byte(secret))
	sum.Write(body)
	return hmac.Equal(sum.Sum(nil), mac)
}

func secretEqual(got, want string) bool {
	if want == "" {
		return false
	}
	sumGot := sha256.Sum256([]byte(got))
	sumWant := sha256.Sum256([]byte(want))
	return subtle.ConstantTimeCompare(sumGot[:], sumWant[:]) == 1
}

func bearerOK(header, token string) bool {
	const prefix = "Bearer "
	value, ok := strings.CutPrefix(header, prefix)
	if !ok || token == "" {
		return false
	}
	return secretEqual(value, token)
}

func validSHA(sha string) bool {
	if len(sha) != 40 && len(sha) != 64 {
		return false
	}
	for _, r := range sha {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f':
		default:
			return false
		}
	}
	return true
}

func zeroSHA(sha string) bool {
	if sha == "" {
		return true
	}
	for _, r := range sha {
		if r != '0' {
			return false
		}
	}
	return true
}

func sign(secret string, body []byte) string {
	sum := hmac.New(sha256.New, []byte(secret))
	sum.Write(body)
	return "sha256=" + hex.EncodeToString(sum.Sum(nil))
}

func (s *Service) DecideHook(event, signature string, body []byte) (int, string, string) {
	if !s.WebhookEnabled() {
		return http.StatusNotFound, "", ""
	}
	if !verifySignature(s.cfg.WebhookSecret, body, signature) {
		return http.StatusUnauthorized, "", ""
	}
	if event == "ping" || event != "push" {
		return http.StatusNoContent, "", ""
	}
	var payload struct {
		Ref        string `json:"ref"`
		After      string `json:"after"`
		Repository struct {
			CloneURL string `json:"clone_url"`
			SSHURL   string `json:"ssh_url"`
			GitURL   string `json:"git_url"`
			HTMLURL  string `json:"html_url"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return http.StatusBadRequest, "", ""
	}
	payload.After = strings.ToLower(payload.After)
	if !validSHA(payload.After) || zeroSHA(payload.After) {
		return http.StatusNoContent, "", ""
	}
	remote, branch, ok := s.hookTarget(payload.Ref, []string{
		payload.Repository.CloneURL,
		payload.Repository.SSHURL,
		payload.Repository.GitURL,
		payload.Repository.HTMLURL,
	})
	if !ok || payload.Ref != "refs/heads/"+branch {
		return http.StatusNoContent, "", ""
	}
	return http.StatusAccepted, remote, payload.After
}

func (s *Service) hookTarget(ref string, urls []string) (string, string, bool) {
	present := false
	for _, item := range urls {
		if strings.TrimSpace(item) != "" {
			present = true
			break
		}
	}
	if s.sourceMode() {
		repos, err := s.discover()
		if err != nil {
			return "", "", false
		}
		for _, repo := range repos {
			if repo.Remote == "" || repo.Branch == "" || ref != "refs/heads/"+repo.Branch {
				continue
			}
			if !present {
				if len(repos) == 1 {
					return repo.Remote, repo.Branch, true
				}
				continue
			}
			for _, item := range urls {
				if sameRemote(repo.Remote, item) {
					return repo.Remote, repo.Branch, true
				}
			}
		}
		return "", "", false
	}
	if present {
		matched := false
		for _, item := range urls {
			if sameRemote(s.cfg.Remote, item) {
				matched = true
				break
			}
		}
		if !matched {
			return "", "", false
		}
	}
	if ref != "refs/heads/"+s.cfg.Branch {
		return "", "", false
	}
	return "", s.cfg.Branch, true
}

func ParseDeploy(body []byte) (string, string, error) {
	if len(strings.TrimSpace(string(body))) == 0 {
		return "", "", nil
	}
	var req struct {
		SHA    string `json:"sha"`
		Remote string `json:"remote"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return "", "", err
	}
	req.SHA = strings.ToLower(req.SHA)
	if req.SHA != "" && !validSHA(req.SHA) {
		return "", "", errInvalidSHA
	}
	return strings.TrimSpace(req.Remote), req.SHA, nil
}

func ParseDeploySHA(body []byte) (string, error) {
	_, sha, err := ParseDeploy(body)
	return sha, err
}

var errInvalidSHA = errors.New("invalid sha")
