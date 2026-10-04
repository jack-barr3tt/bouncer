package access

import "sync"

type Event struct {
	Name string
	Apps []string
}

type Broker struct {
	mu   sync.Mutex
	subs map[string]map[chan Event]struct{}
}

func New() *Broker {
	return &Broker{subs: map[string]map[chan Event]struct{}{}}
}

func (b *Broker) Subscribe(key string) (<-chan Event, func()) {
	ch := make(chan Event, 4)
	b.mu.Lock()
	if b.subs[key] == nil {
		b.subs[key] = map[chan Event]struct{}{}
	}
	b.subs[key][ch] = struct{}{}
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		delete(b.subs[key], ch)
		if len(b.subs[key]) == 0 {
			delete(b.subs, key)
		}
		b.mu.Unlock()
	}
}

func (b *Broker) Publish(key string, ev Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs[key] {
		select {
		case ch <- ev:
		default:
		}
	}
}
