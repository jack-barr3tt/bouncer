package deploy

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type memStore struct {
	mu   sync.Mutex
	runs []memRun
}

type memRun struct {
	id       uuid.UUID
	sha      string
	status   string
	log      string
	started  time.Time
	finished *time.Time
}

func (m *memStore) Start(ctx context.Context, sha string) (uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := uuid.New()
	m.runs = append(m.runs, memRun{id: id, sha: sha, status: "running", started: time.Now().UTC()})
	return id, nil
}

func (m *memStore) Finish(ctx context.Context, id uuid.UUID, sha, status, logText string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	for i := range m.runs {
		if m.runs[i].id == id {
			m.runs[i].sha = sha
			m.runs[i].status = status
			m.runs[i].log = logText
			m.runs[i].finished = &now
			return nil
		}
	}
	return errInvalidSHA
}

func (m *memStore) LastPublished(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var sha string
	var latest time.Time
	for _, run := range m.runs {
		if run.status != "published" || run.finished == nil {
			continue
		}
		if sha == "" || run.finished.After(latest) {
			sha = run.sha
			latest = *run.finished
		}
	}
	return sha, nil
}

func (m *memStore) Latest(ctx context.Context) (*Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.runs) == 0 {
		return nil, nil
	}
	run := m.runs[len(m.runs)-1]
	return &Run{SHA: run.sha, Status: run.status, Log: run.log, StartedAt: run.started, FinishedAt: run.finished}, nil
}

func (m *memStore) FailRunning(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	for i := range m.runs {
		if m.runs[i].status != "running" {
			continue
		}
		m.runs[i].status = "failed"
		m.runs[i].finished = &now
		if m.runs[i].log == "" {
			m.runs[i].log = "interrupted"
		} else {
			m.runs[i].log += "\ninterrupted"
		}
	}
	return nil
}

func (m *memStore) seed(sha string) {
	now := time.Now().UTC()
	m.runs = append(m.runs, memRun{id: uuid.New(), sha: sha, status: "published", started: now, finished: &now})
}
