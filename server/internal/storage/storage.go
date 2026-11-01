package storage

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

const DefaultMaxBytes = 1 << 20

var (
	ErrNotFound = errors.New("storage object not found")
	ErrConflict = errors.New("storage object changed")
)

type Object struct {
	Body     []byte
	Revision string
}

type Store interface {
	Get(ctx context.Context, key string) (Object, error)
	Put(ctx context.Context, key string, body []byte, revision string) (string, error)
	Delete(ctx context.Context, key string) error
	DeletePrefix(ctx context.Context, prefix string) error
}

func UserKey(id uuid.UUID, slug string) string {
	return "users/" + id.String() + "/" + slug + ".json"
}

func TempPrefix(id uuid.UUID) string {
	return "temporary/" + id.String() + "/"
}

func TempKey(id uuid.UUID, slug string) string {
	return TempPrefix(id) + slug + ".json"
}

func ParseMaxBytes(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultMaxBytes, nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 || n > math.MaxInt {
		return 0, fmt.Errorf("STORAGE_MAX_BYTES must be a positive integer")
	}
	return int(n), nil
}
