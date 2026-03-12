package webapp

import (
	"context"
	"sync"
	"time"
)

// PubkeyCollector collects temp ECDH pubkeys for a task.
// It is used to replace polling-based "wait for all pubkeys" logic with event-driven waiting.
type PubkeyCollector struct {
	mu         sync.RWMutex
	expected   map[string]bool
	submitted  map[string][]byte // subject -> pubkey bytes
	completed  bool
	completeCh chan struct{} // notify completion (buffered to avoid blocking)
}

func NewPubkeyCollector(subjects []string) *PubkeyCollector {
	expected := make(map[string]bool, len(subjects))
	for _, s := range subjects {
		expected[s] = true
	}
	return &PubkeyCollector{
		expected:   expected,
		submitted:  make(map[string][]byte, len(subjects)),
		completeCh: make(chan struct{}, 1),
	}
}

// Submit stores pubkey for subject. Returns true if this submit completes the set.
func (c *PubkeyCollector) Submit(subject string, pubkey []byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.completed || !c.expected[subject] {
		return false
	}
	c.submitted[subject] = pubkey

	if len(c.submitted) == len(c.expected) {
		c.completed = true
		select {
		case c.completeCh <- struct{}{}:
		default:
		}
		return true
	}
	return false
}

func (c *PubkeyCollector) GetPubkeys() map[string][]byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make(map[string][]byte, len(c.submitted))
	for k, v := range c.submitted {
		res[k] = v
	}
	return res
}

func (c *PubkeyCollector) Wait(ctx context.Context) error {
	c.mu.RLock()
	if c.completed {
		c.mu.RUnlock()
		return nil
	}
	ch := c.completeCh
	c.mu.RUnlock()

	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

var (
	pubkeyCollectorsMu sync.RWMutex
	pubkeyCollectors   = make(map[string]*PubkeyCollector) // key = module|taskID
)

func pubkeyCollectorKey(module, taskID string) string { return module + "|" + taskID }

func registerPubkeyCollector(module, taskID string, c *PubkeyCollector, ttl time.Duration) {
	key := pubkeyCollectorKey(module, taskID)
	pubkeyCollectorsMu.Lock()
	pubkeyCollectors[key] = c
	pubkeyCollectorsMu.Unlock()

	time.AfterFunc(ttl, func() {
		unregisterPubkeyCollector(module, taskID)
	})
}

func unregisterPubkeyCollector(module, taskID string) {
	pubkeyCollectorsMu.Lock()
	delete(pubkeyCollectors, pubkeyCollectorKey(module, taskID))
	pubkeyCollectorsMu.Unlock()
}

func submitPubkeyToCollector(module, taskID, subject string, pubkey []byte) {
	pubkeyCollectorsMu.RLock()
	c := pubkeyCollectors[pubkeyCollectorKey(module, taskID)]
	pubkeyCollectorsMu.RUnlock()
	if c == nil {
		return
	}
	c.Submit(subject, pubkey)
}
