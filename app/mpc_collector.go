package app

import (
	"context"
	"sync"
	"time"

	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
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

// --- Sign result collector ---

// SignResultCollector collects sign results per subject for a task.
type SignResultCollector struct {
	mu         sync.RWMutex
	expected   map[string]bool
	submitted  map[string]*dto.CliMPCSignResultReq // subject -> result
	completed  bool
	completeCh chan struct{}
}

func NewSignResultCollector(subjects []string) *SignResultCollector {
	expected := make(map[string]bool, len(subjects))
	for _, s := range subjects {
		expected[s] = true
	}
	return &SignResultCollector{
		expected:   expected,
		submitted:  make(map[string]*dto.CliMPCSignResultReq, len(subjects)),
		completeCh: make(chan struct{}, 1),
	}
}

func (c *SignResultCollector) Submit(subject string, res *dto.CliMPCSignResultReq) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.completed || !c.expected[subject] || res == nil {
		return false
	}
	c.submitted[subject] = res

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

func (c *SignResultCollector) GetResults() map[string]*dto.CliMPCSignResultReq {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]*dto.CliMPCSignResultReq, len(c.submitted))
	for k, v := range c.submitted {
		out[k] = v
	}
	return out
}

func (c *SignResultCollector) Wait(ctx context.Context) error {
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
	signResultCollectorsMu sync.RWMutex
	signResultCollectors   = make(map[string]*SignResultCollector) // key = taskID
)

func registerSignResultCollector(taskID string, c *SignResultCollector, ttl time.Duration) {
	signResultCollectorsMu.Lock()
	signResultCollectors[taskID] = c
	signResultCollectorsMu.Unlock()

	time.AfterFunc(ttl, func() {
		unregisterSignResultCollector(taskID)
	})
}

func unregisterSignResultCollector(taskID string) {
	signResultCollectorsMu.Lock()
	delete(signResultCollectors, taskID)
	signResultCollectorsMu.Unlock()
}

func submitSignResultToCollector(taskID, subject string, res *dto.CliMPCSignResultReq) {
	signResultCollectorsMu.RLock()
	c := signResultCollectors[taskID]
	signResultCollectorsMu.RUnlock()
	if c == nil {
		return
	}
	c.Submit(subject, res)
}

// --- Keygen result collector ---

// KeygenResultCollector collects keygen node results per subject for a task.
type KeygenResultCollector struct {
	mu         sync.RWMutex
	expected   map[string]bool
	submitted  map[string]*MpcKeygenNodeResult // subject -> node result (Status=40)
	completed  bool
	completeCh chan struct{}
}

func NewKeygenResultCollector(subjects []string) *KeygenResultCollector {
	expected := make(map[string]bool, len(subjects))
	for _, s := range subjects {
		expected[s] = true
	}
	return &KeygenResultCollector{
		expected:   expected,
		submitted:  make(map[string]*MpcKeygenNodeResult, len(subjects)),
		completeCh: make(chan struct{}, 1),
	}
}

func (c *KeygenResultCollector) Submit(subject string, res *MpcKeygenNodeResult) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.completed || !c.expected[subject] || res == nil || res.Status != 40 {
		return false
	}
	c.submitted[subject] = res

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

func (c *KeygenResultCollector) GetResults() map[string]*MpcKeygenNodeResult {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]*MpcKeygenNodeResult, len(c.submitted))
	for k, v := range c.submitted {
		out[k] = v
	}
	return out
}

func (c *KeygenResultCollector) Wait(ctx context.Context) error {
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
	keygenResultCollectorsMu sync.RWMutex
	keygenResultCollectors   = make(map[string]*KeygenResultCollector) // key = taskID
)

func registerKeygenResultCollector(taskID string, c *KeygenResultCollector, ttl time.Duration) {
	keygenResultCollectorsMu.Lock()
	keygenResultCollectors[taskID] = c
	keygenResultCollectorsMu.Unlock()

	time.AfterFunc(ttl, func() {
		unregisterKeygenResultCollector(taskID)
	})
}

func unregisterKeygenResultCollector(taskID string) {
	keygenResultCollectorsMu.Lock()
	delete(keygenResultCollectors, taskID)
	keygenResultCollectorsMu.Unlock()
}

func submitKeygenResultToCollector(taskID, subject string, res *MpcKeygenNodeResult) {
	keygenResultCollectorsMu.RLock()
	c := keygenResultCollectors[taskID]
	keygenResultCollectorsMu.RUnlock()
	if c == nil {
		return
	}
	c.Submit(subject, res)
}
