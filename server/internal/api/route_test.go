package api

import "testing"

func TestSubdomainModeIsIgnoredForAnIP(t *testing.T) {
	route, err := parseRouting("http://127.0.0.1:8080", "subdomain")
	if err != nil {
		t.Fatal(err)
	}
	if route.mode != "path" {
		t.Fatalf("mode: %s", route.mode)
	}
}

func TestRoutingRejectsAnUnknownMode(t *testing.T) {
	if _, err := parseRouting("https://example.com", "host"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestStripPrefix(t *testing.T) {
	if got := stripPrefix("/apps/notes/items", "/apps/notes"); got != "/items" {
		t.Fatalf("path: %s", got)
	}
	if got := stripPrefix("/apps/notes", "/apps/notes"); got != "/" {
		t.Fatalf("root: %s", got)
	}
	if got := stripPrefix("/apps/notes-other", "/apps/notes"); got != "/apps/notes-other" {
		t.Fatalf("kept: %s", got)
	}
}
