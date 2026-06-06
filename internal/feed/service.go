// internal/feed/service.go
package feed

import (
	"context"
	"log/slog"
	"sync"

	"github.com/Master-Daniels/marrow/internal/config"
	"github.com/Master-Daniels/marrow/internal/domain"
	"github.com/Master-Daniels/marrow/internal/poller"
	"github.com/Master-Daniels/marrow/internal/storage"
)

// Service manages all active pollers and provides a broadcast channel for new items.
type Service struct {
	mu        sync.RWMutex
	pollers   map[string]*poller.Poller // key = source ID
	notifyCh  chan *domain.FeedItem     // SSESender reads from this
	store     storage.Repository
	logger    *slog.Logger
	appconfig *config.AppConfig
	ctx       context.Context
	cancel    context.CancelFunc
}

// New creates the service. Call `Start` after you have loaded the config.
func New(parent context.Context, store storage.Repository, logger *slog.Logger, appconfig *config.AppConfig) *Service {
	ctx, cancel := context.WithCancel(parent)
	return &Service{
		pollers:   make(map[string]*poller.Poller),
		notifyCh:  make(chan *domain.FeedItem, 100), // buffered, drop oldest if full
		store:     store,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
		appconfig: appconfig,
	}
}

// StartAll creates pollers for the supplied sources and runs them.
func (s *Service) StartAll(srcs []config.SourceDef) {
	s.SyncSources(srcs)
}

// SyncSources compares running pollers with the incoming configuration,
// persisting the source list to the database, then starting, stopping,
// or updating pollers as necessary.
func (s *Service) SyncSources(srcs []config.SourceDef) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("synchronizing source configurations...")

	// Persist the incoming source list and remove any stale persisted sources.
	incomingSourceIDs := make(map[string]struct{}, len(srcs))
	for _, src := range srcs {
		incomingSourceIDs[src.ID] = struct{}{}
	}

	persistedSources, err := s.store.GetAllSources(s.ctx)
	if err != nil {
		s.logger.Error("failed to load persisted sources", "error", err)
		return
	}

	for _, persisted := range persistedSources {
		if _, exists := incomingSourceIDs[persisted.ID]; !exists {
			s.logger.Info("persisted source no longer in config, removing", "source", persisted.ID)
			if err := s.store.RemoveSource(s.ctx, persisted.ID); err != nil {
				s.logger.Error("failed to remove stale persisted source", "source", persisted.ID, "error", err)
				return
			}
		}
	}

	for _, src := range srcs {
		domainSource := sourceDefToDomain(src)
		if err := s.store.AddSource(s.ctx, domainSource); err != nil {
			s.logger.Error("failed to persist source", "source", src.ID, "error", err)
			return
		}
	}

	// Create a map of incoming sources for O(1) lookups
	newSrcs := make(map[string]config.SourceDef, len(srcs))
	for _, src := range srcs {
		newSrcs[src.ID] = src
	}

	// 1. Stop and remove pollers that are no longer in the config,
	// or those whose critical fields have changed.
	for id, p := range s.pollers {
		newDef, exists := newSrcs[id]
		if !exists {
			s.logger.Info("source removed, stopping poller", "source", id)
			p.Stop()
			delete(s.pollers, id)
			continue
		}

		// If URL or PollInterval changed, we must recreate the poller
		oldDef := p.Config()
		if oldDef.FeedURL != newDef.FeedURL || oldDef.PollInterval != newDef.PollInterval {
			s.logger.Info("source configuration changed, restarting poller", "source", id)
			p.Stop()
			delete(s.pollers, id)
		}
	}

	// 2. Start new pollers for added or updated sources
	for id, src := range newSrcs {
		if _, running := s.pollers[id]; !running {
			p := poller.New(s.ctx, src, s.store, s.logger, s.notifyCh, s.appconfig.RSSHubURL)
			s.pollers[id] = p
			p.Start()
			s.logger.Info("poller started", "source", id)
		}
	}
}

// StopAll gracefully stops every poller.
func (s *Service) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, p := range s.pollers {
		p.Stop()
		delete(s.pollers, id)
		s.logger.Info("poller stopped", "source", id)
	}
	s.cancel()
}

// Subscribe returns a read‑only channel that receives every new FeedItem
// as soon as the poller inserts it into the DB.  The channel is closed when
// the Service is stopped.
func (s *Service) Subscribe() <-chan *domain.FeedItem {
	return s.notifyCh
}
