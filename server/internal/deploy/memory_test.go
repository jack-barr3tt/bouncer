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
	remote   string
	sha      string
	status   string
	log      string
	started  time.Time
	finished *time.Time
}

func (m *memStore) Start(ctx context.Context, remote, sha string) (uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := uuid.New()
	m.runs = append(m.runs, memRun{id: id, remote: remote, sha: sha, status: "running", started: time.Now().UTC()})
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

func (m *memStore) LastPublished(ctx context.Context, remote string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var sha string
	var latest time.Time
	for _, run := range m.runs {
		if run.remote != remote || run.status != "published" || run.sha == "" || run.finished == nil {
			continue
		}
		if sha == "" || !run.finished.Before(latest) {
			sha = run.sha
			latest = *run.finished
		}
	}
	return sha, nil
}

func (m *memStore) Latest(ctx context.Context, remote string) (*Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var found *memRun
	for i := range m.runs {
		run := &m.runs[i]
		if run.remote != remote {
			continue
		}
		if found == nil || !run.started.Before(found.started) {
			found = run
		}
	}
	if found == nil {
		return nil, nil
	}
	return &Run{SHA: found.sha, Status: found.status, Log: found.log, StartedAt: found.started, FinishedAt: found.finished}, nil
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

func (m *memStore) seed(remote, sha string) {
	now := time.Now().UTC()
	m.runs = append(m.runs, memRun{id: uuid.New(), remote: remote, sha: sha, status: "published", started: now, finished: &now})
}
