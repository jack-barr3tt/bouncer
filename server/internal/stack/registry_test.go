package stack

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLatestVersionIgnoresNonReleases(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			_, _ = w.Write([]byte(`{"token":"t"}`))
		case "/v2/jack-barr3tt/bouncer/tags/list":
			if r.Header.Get("Authorization") != "Bearer t" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"tags":["latest","0.2.0","0.10.0","0.9.1","dev"]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := srv.Client()
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	client.Transport = roundTrip(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = srv.Listener.Addr().String()
		return base.RoundTrip(req)
	})
	got, err := latestVersion(context.Background(), client, "ghcr.io", "jack-barr3tt/bouncer")
	if err != nil {
		t.Fatal(err)
	}
	if got != "0.10.0" {
		t.Fatalf("latest: %s", got)
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (r roundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return r(req) }
