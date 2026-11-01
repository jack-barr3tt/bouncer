package api

import "testing"

func TestJSONObject(t *testing.T) {
	for _, body := range []string{`{}`, ` {"a":1} `, "{\n\"a\": 1\n}"} {
		if !jsonObject([]byte(body)) {
			t.Fatalf("rejected %q", body)
		}
	}
	for _, body := range []string{"", "null", "[]", "1", `"hi"`, `{"a":1}[]`, `{"a":1} nope`, `{`, `{"a":`} {
		if jsonObject([]byte(body)) {
			t.Fatalf("accepted %q", body)
		}
	}
}
