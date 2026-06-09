package poller

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Master-Daniels/marrow/internal/config"
	"github.com/Master-Daniels/marrow/internal/domain"
	marrowErrors "github.com/Master-Daniels/marrow/internal/errors"
	"github.com/Master-Daniels/marrow/internal/parser"
	"github.com/Master-Daniels/marrow/internal/storage"
)

// BackOffConfig holds the exponential‑back‑off parameters.
// You can expose these via env/config later if you want.
type BackOffConfig struct {
	BaseDelay   time.Duration // first retry after this delay
	MaxDelay    time.Duration // ceiling
	Multiplier  float64       // 2.0 = double each attempt
	MaxAttempts int           // 0 = unlimited
}

// defaultBackOff is used if the caller passes a zero value.
var defaultBackOff = BackOffConfig{
	BaseDelay:   30 * time.Second,
	MaxDelay:    5 * time.Minute,
	Multiplier:  2.0,
	MaxAttempts: 0,
}

// Poller handles a single source.
type Poller struct {
	rsshubURL  string
	src        config.SourceDef
	client     *http.Client
	parser     *parser.Parser
	store      storage.Repository
	logger     *slog.Logger
	backoff    BackOffConfig
	failCount  int
	ctx        context.Context
	cancel     context.CancelFunc
	notifyChan chan<- *domain.FeedItem // optional – used by feed.Service
}

// New creates a poller for one source. The caller supplies a ctx that will be
// the parent for the poller’s own cancellable context.
func New(parentCtx context.Context, src config.SourceDef,
	store storage.Repository, logger *slog.Logger,
	notifyChan chan<- *domain.FeedItem, rsshubURL string) *Poller {

	ctx, cancel := context.WithCancel(parentCtx)

	// http client with a modest timeout – individual requests can be
	// overridden later with request‑scoped timeouts.
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	return &Poller{
		src:        src,
		client:     client,
		parser:     parser.New(),
		store:      store,
		logger:     logger.With("source", src.ID),
		backoff:    defaultBackOff,
		ctx:        ctx,
		cancel:     cancel,
		rsshubURL:  rsshubURL,
		notifyChan: notifyChan,
	}
}

// ---- Public API ---------------------------------------------------------

// Start begins the ticker loop. It returns immediately; the work runs in a
// background goroutine.
func (p *Poller) Start() {
	go p.run()
}

// Stop signals the goroutine to exit and waits for it to finish.
func (p *Poller) Stop() {
	p.cancel()
}

// Config returns the current source configuration used by this poller.
func (p *Poller) Config() config.SourceDef {
	return p.src
}

// ---- Internal ----------------------------------------------------------------
func (p *Poller) fetchURL() string {
	if p.rsshubURL == "" || !p.shouldUseRSSHub(p.src.FeedURL) {
		return p.src.FeedURL
	}
	return fmt.Sprintf("%s/%s",
		strings.TrimRight(p.rsshubURL, "/"),
		p.src.FeedURL)
}

func (p *Poller) shouldUseRSSHub(raw string) bool {
	if p.src.UseRSSHub {
		return true
	}

	u, err := url.Parse(raw)
	if err != nil {
		return true
	}
	path := strings.ToLower(u.Path)
	if strings.HasPrefix(path, "https://") ||
		strings.HasSuffix(path, ".xml") ||
		strings.HasSuffix(path, ".rss") ||
		strings.HasSuffix(path, ".atom") ||
		strings.Contains(path, "/feed") ||
		strings.Contains(path, "/rss") {
		return false
	}
	return true
}
func (p *Poller) run() {
	// Parse the user‑provided interval; fallback to 15m.
	interval, err := time.ParseDuration(p.src.PollInterval)
	if err != nil || interval <= 0 {
		interval = 15 * time.Minute
	}
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-p.ctx.Done():
			p.logger.Info("poller stopped")
			return
		case <-timer.C:
			nextDelay := p.fetchOnce(interval)
			timer.Reset(nextDelay)
		}
	}
}

// fetchOnce performs a single HTTP GET → parse → store cycle.
// It also handles back‑off on error.
func (p *Poller) fetchOnce(normalInterval time.Duration) time.Duration {
	if p.backoffReachedLimit() {
		// We have exhausted attempts (if MaxAttempts > 0)
		p.logger.Warn("max back‑off attempts reached, skipping this tick")
		return normalInterval // return to normal interval
	}

	p.logger.Debug("fetching feed")
	ctx, cancel := context.WithTimeout(p.ctx, 20*time.Second)
	defer cancel()

	targetURL := p.fetchURL()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return p.handleError(fmt.Errorf("request creation: %w", err))

	}
	// optional custom headers from config
	for k, v := range p.src.Headers {
		req.Header.Set(k, v)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return p.handleError(fmt.Errorf("http error: %w", err))
		
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return p.handleError(fmt.Errorf("unexpected status %d", resp.StatusCode))
		
	}

	items, err := p.parser.Parse(resp.Body, p.src)
	if err != nil {
		return p.handleError(marrowErrors.NewParsingError(p.src.ID, "failed to parse feed format", err))
		
	}

	// Insert each item; deduplication is handled by the DB (INSERT OR IGNORE)
	// but we also compute a deterministic hash to use as the primary key.
	var inserted int
	for _, it := range items {
		dedupItem := DeduplicateItem{
			GUID:        it.ID,
			Link:        it.Link,
			Title:       it.Title,
			Source:      it.SourceID,
			Description: it.Description,
			PubDate:     it.PublishedAt,
			LastUpdated: time.Now(),
		}
		dedupedID, err := deduplicateFunction(dedupItem)
		if err != nil {
			p.logger.Error("deduplication error", "err", err)
			return normalInterval
		}
		it.ID = dedupedID
		it.SourceID = p.src.ID
		if err := p.store.Insert(p.ctx, it); err != nil {
			p.logger.Error("store error", "err", err)
			continue
		}
		inserted++
		// fire notification (non-blocking)
		if p.notifyChan != nil {
			select {
			case p.notifyChan <- it:
			default: // drop if nobody is listening
			}

		}
	}

	if inserted > 0 {
		p.logger.Info("fetched new items", "count", inserted)
	}
	p.handleSuccess()
	return normalInterval
}

// handleError records the failure, increments the back‑off counter and
// optionally logs the error message.
func (p *Poller) handleError(err error) time.Duration {
	p.failCount++

	// Structured logging with contextual information.
	p.logger.Error("poll error", "err", err, "failCount", p.failCount, "url", p.fetchURL())

	// Fetch previous state to preserve the last successful time
	lastSuccess := time.Time{}
	if existing, err := p.store.GetPollingState(p.ctx, p.src.ID); err == nil && existing != nil {
		lastSuccess = existing.LastSuccessfulPoll
	}

	// Update DB state
	state := &domain.PollingState{
		SourceID:            p.src.ID,
		LastSuccessfulPoll:  lastSuccess,
		ConsecutiveFailures: p.failCount,
		LastError:           err.Error(),
		LastPolledAt:        time.Now(),
	}
	if dbErr := p.store.SaveOrUpdatePollingState(p.ctx, state); dbErr != nil {
		p.logger.Error("failed to persist error state to db", "err", dbErr)
	}

	delay := p.computeBackoff()
	p.logger.Debug("applying back‑off", "delay", delay)
	return delay
}

// handleSuccess resets the failure counter and back‑off state.
func (p *Poller) handleSuccess() {
	if p.failCount > 0 {
		p.logger.Info("poll succeeded after failures", "prevFails", p.failCount)
	}
	p.failCount = 0

	state := &domain.PollingState{
		SourceID:            p.src.ID,
		LastSuccessfulPoll:  time.Now(),
		ConsecutiveFailures: 0,
		LastError:           "",
		LastPolledAt:        time.Now(),
	}

	if dbErr := p.store.SaveOrUpdatePollingState(p.ctx, state); dbErr != nil {
		p.logger.Error("failed to persist success state to db", "err", dbErr)
	}
}

// backoffReachedLimit returns true when MaxAttempts is >0 and we have reached it.
func (p *Poller) backoffReachedLimit() bool {
	return p.backoff.MaxAttempts > 0 && p.failCount >= p.backoff.MaxAttempts
}

// computeBackoff returns the amount of time we should wait before the next
// attempt, based on the current failure count.
func (p *Poller) computeBackoff() time.Duration {
	if p.failCount == 0 {
		return 0
	}
	// Check for overflow and return max delay
	mul := 1.0
	delay := p.backoff.BaseDelay
	for i := 0; i < p.failCount-1; i++ {
		mul *= p.backoff.Multiplier
		delay = time.Duration(float64(p.backoff.BaseDelay) * mul)
		if delay > p.backoff.MaxDelay {
			return p.backoff.MaxDelay
		}
	}
	return delay
}
