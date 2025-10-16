package internal

import (
	"context"
	"log"
	"sync"
	"time"
)

type Manager struct {
	cfg   *Config
	store *FileStore

	mu       sync.Mutex
	running  map[string]bool
	breakers map[string]*CircuitBreaker

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewManager(cfg *Config, st *FileStore) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		cfg:      cfg,
		store:    st,
		running:  map[string]bool{},
		breakers: map[string]*CircuitBreaker{},
		ctx:      ctx,
		cancel:   cancel,
	}
}

// breakerFor - thread-safe version for public use
func (m *Manager) breakerFor(app AppConfig) *CircuitBreaker {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.breakerForUnsafe(app)
}

// breakerForUnsafe - internal version, MUST be called with m.mu already acquired
func (m *Manager) breakerForUnsafe(app AppConfig) *CircuitBreaker {
	k := app.AppID + "-" + app.Country
	if b, ok := m.breakers[k]; ok {
		return b
	}
	b := NewCircuitBreaker(
		m.cfg.CircuitBreaker.FailureThreshold,
		time.Duration(m.cfg.CircuitBreaker.OpenCooldownSeconds)*time.Second,
	)
	m.breakers[k] = b
	return b
}

// Start launches workers; now uses manager's context
func (m *Manager) Start() {
	for _, app := range m.cfg.Apps {
		app := app
		m.wg.Add(1)
		go m.worker(m.ctx, app)
	}
}

// Stop shuts down everything with ctx
func (m *Manager) Stop() {
	m.cancel()
	m.wg.Wait()
}

func (m *Manager) worker(ctx context.Context, app AppConfig) {
	defer m.wg.Done()

	interval := time.Duration(m.cfg.PollIntervalMinutes) * time.Minute
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// start immediately
	m.PollOnce(ctx, app)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.PollOnce(ctx, app)
		}
	}
}

func (m *Manager) PollOnce(ctx context.Context, app AppConfig) {
	k := app.AppID + "-" + app.Country

	m.mu.Lock()
	if m.running[k] {
		m.mu.Unlock()
		return
	}
	m.running[k] = true
	cb := m.breakerForUnsafe(app)
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.running[k] = false
		m.mu.Unlock()
	}()

	if !cb.Allow() {
		log.Printf("[poll %s] circuit breaker OPEN (%s), skipping iteration", k, cb.State())
		return
	}

	seen := m.store.GetSeenSet(app.AppID, app.Country)

	newTotal := 0
	newIDs := []string{}
	toAppend := []Review{}

	const maxPages = 10
	for page := 1; page <= maxPages; page++ {
		// Per-page context (to avoid long blocks)
		pageCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		revs, err := FetchPageWithRetry(pageCtx, m.cfg, app.Country, app.AppID, page)
		cancel()

		if err != nil {
			// ITERATION FAILURE: report to CB and break
			cb.Failure()
			log.Printf("[poll %s] fetch page %d failed after retries: %v", k, page, err)
			return
		}
		// success on this page → report to CB
		cb.Success()

		if len(revs) == 0 {
			break
		}

		pageNew := 0
		for _, r := range revs {
			if _, ok := seen[r.ID]; ok {
				continue
			}
			// Consider all reviews (48h filter will be in GET /reviews)
			seen[r.ID] = struct{}{}
			pageNew++
			newTotal++
			newIDs = append(newIDs, r.ID)
			toAppend = append(toAppend, r)
		}

		// Early stop: if no new items found here, older ones follow
		if pageNew == 0 {
			break
		}

		// Short pause between pages for rate limiting
		time.Sleep(300 * time.Millisecond)
	}

	if newTotal > 0 {
		if err := m.store.AppendReviews(app.AppID, app.Country, toAppend, newIDs); err != nil {
			log.Printf("[poll %s] append error: %v", k, err)
		} else {
			log.Printf("[poll %s] appended %d new reviews", k, newTotal)
		}
	} else {
		// update lastPoll only
		_ = m.store.AppendReviews(app.AppID, app.Country, nil, nil)
		log.Printf("[poll %s] no new reviews", k)
	}
}

func (m *Manager) Apps() []AppConfig { return m.cfg.Apps }
