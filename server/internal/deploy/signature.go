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

func (s *Service) DecideHook(event, signature string, body []byte) (int, string) {
	if !s.WebhookEnabled() {
		return http.StatusNotFound, ""
	}
	if !verifySignature(s.cfg.WebhookSecret, body, signature) {
		return http.StatusUnauthorized, ""
	}
	if event == "ping" || event != "push" {
		return http.StatusNoContent, ""
	}
	var payload struct {
		Ref   string `json:"ref"`
		After string `json:"after"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return http.StatusBadRequest, ""
	}
	payload.After = strings.ToLower(payload.After)
	if payload.Ref != "refs/heads/"+s.cfg.Branch {
		return http.StatusNoContent, ""
	}
	if !validSHA(payload.After) || zeroSHA(payload.After) {
		return http.StatusNoContent, ""
	}
	return http.StatusAccepted, payload.After
}

func ParseDeploySHA(body []byte) (string, error) {
	if len(strings.TrimSpace(string(body))) == 0 {
		return "", nil
	}
	var req struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return "", err
	}
	req.SHA = strings.ToLower(req.SHA)
	if req.SHA != "" && !validSHA(req.SHA) {
		return "", errInvalidSHA
	}
	return req.SHA, nil
}

var errInvalidSHA = errors.New("invalid sha")
