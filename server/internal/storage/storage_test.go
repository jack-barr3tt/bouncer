package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestParseMaxBytes(t *testing.T) {
	got, err := ParseMaxBytes("")
	if err != nil || got != DefaultMaxBytes {
		t.Fatalf("default: %d %v", got, err)
	}
	got, err = ParseMaxBytes(" 32 ")
	if err != nil || got != 32 {
		t.Fatalf("set: %d %v", got, err)
	}
	for _, raw := range []string{"0", "-1", "nope", "1.5"} {
		if _, err := ParseMaxBytes(raw); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
}

func TestMemoryRevisions(t *testing.T) {
	ctx := context.Background()
	mem := NewMemory()
	key := UserKey(uuid.MustParse("11111111-1111-1111-1111-111111111111"), "hello")
	if _, err := mem.Get(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	rev, err := mem.Put(ctx, key, []byte(`{"n":1}`), "")
	if err != nil || rev == "" {
		t.Fatalf("create: %q %v", rev, err)
	}
	if _, err := mem.Put(ctx, key, []byte(`{"n":2}`), ""); !errors.Is(err, ErrConflict) {
		t.Fatalf("second create: %v", err)
	}
	if _, err := mem.Put(ctx, key, []byte(`{"n":2}`), "stale"); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale: %v", err)
	}
	next, err := mem.Put(ctx, key, []byte(`{"n":2}`), rev)
	if err != nil || next == rev {
		t.Fatalf("update: %q %v", next, err)
	}
	obj, err := mem.Get(ctx, key)
	if err != nil || string(obj.Body) != `{"n":2}` || obj.Revision != next {
		t.Fatalf("get: %+v %v", obj, err)
	}
	other := TempKey(uuid.MustParse("22222222-2222-2222-2222-222222222222"), "hello")
	if _, err := mem.Put(ctx, other, []byte(`{}`), ""); err != nil {
		t.Fatal(err)
	}
	if err := mem.DeletePrefix(ctx, TempPrefix(uuid.MustParse("22222222-2222-2222-2222-222222222222"))); err != nil {
		t.Fatal(err)
	}
	if _, err := mem.Get(ctx, other); !errors.Is(err, ErrNotFound) {
		t.Fatalf("prefix delete: %v", err)
	}
	if _, err := mem.Get(ctx, key); err != nil {
		t.Fatalf("other key removed: %v", err)
	}
	if err := mem.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if err := mem.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
}
