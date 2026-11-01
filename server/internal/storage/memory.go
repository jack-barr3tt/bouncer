package storage

import (
	"context"
	"strconv"
	"sync"
)

type Memory struct {
	mu   sync.Mutex
	objs map[string]object
	seq  int
}

type object struct {
	body     []byte
	revision string
}

func NewMemory() *Memory {
	return &Memory{objs: map[string]object{}}
}

func (m *Memory) Get(_ context.Context, key string) (Object, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	obj, ok := m.objs[key]
	if !ok {
		return Object{}, ErrNotFound
	}
	return Object{Body: append([]byte(nil), obj.body...), Revision: obj.revision}, nil
}

func (m *Memory) Put(_ context.Context, key string, body []byte, revision string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.objs[key]
	if revision == "" {
		if ok {
			return "", ErrConflict
		}
	} else if !ok || cur.revision != revision {
		return "", ErrConflict
	}
	m.seq++
	next := strconv.Itoa(m.seq)
	m.objs[key] = object{body: append([]byte(nil), body...), revision: next}
	return next, nil
}

func (m *Memory) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.objs, key)
	return nil
}

func (m *Memory) DeletePrefix(_ context.Context, prefix string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key := range m.objs {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(m.objs, key)
		}
	}
	return nil
}
