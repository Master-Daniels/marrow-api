package storage

import (
	"context"
	"time"

	"github.com/Master-Daniels/marrow/internal/domain"
)

type PaginationParams struct {
	Limit  int    // Number of items per page
	Offset int    // Number of items to skip
	Cursor string // Optional cursor for cursor-based pagination
}

// Repository defines the interface for data storage operations related to feed items.
type Repository interface {
	// Insert a new feed item. Should handle duplicates gracefully (e.g. ignore or update).
	Insert(ctx context.Context, item *domain.FeedItem) error
	// Get all items, ordered by PublishedAt desc
	All(ctx context.Context, params PaginationParams) ([]*domain.FeedItem, error)
	// Get items by source ID, ordered by PublishedAt desc
	GetBySource(ctx context.Context, sourceID string, params PaginationParams) ([]*domain.FeedItem, error)
	// Get items published after a certain time, ordered by PublishedAt desc
	GetAfter(ctx context.Context, after time.Time, params PaginationParams) ([]*domain.FeedItem, error)
	// Get items by type, ordered by PublishedAt desc
	GetByType(ctx context.Context, contentType domain.ContentType, params PaginationParams) ([]*domain.FeedItem, error)
	// Get items by source ID and type, ordered by PublishedAt desc
	GetBySourceAndType(ctx context.Context, sourceID string, contentType domain.ContentType, params PaginationParams) ([]*domain.FeedItem, error)
	// Get items published after a certain time, ordered by PublishedAt desc
	GetAfterAndType(ctx context.Context, after time.Time, contentType domain.ContentType, params PaginationParams) ([]*domain.FeedItem, error)
	// Get items published after a certain time, ordered by PublishedAt desc
	GetAfterAndSource(ctx context.Context, after time.Time, sourceID string, params PaginationParams) ([]*domain.FeedItem, error)
	// Get the latest N items, ordered by PublishedAt desc
	Latest(ctx context.Context, limit int) ([]*domain.FeedItem, error)
	// Optional: cleanup old items, etc.
	Cleanup(ctx context.Context) error

	// SaveOrUpdatePollingState records the current status of a poller's execution
	SaveOrUpdatePollingState(ctx context.Context, state *domain.PollingState) error

	// GetPollingState returns the current polling state for a given source
	GetPollingState(ctx context.Context, sourceID string) (*domain.PollingState, error)

	// Source persistence and lookups
	GetAllSources(ctx context.Context) ([]*domain.Source, error)
	GetSource(ctx context.Context, sourceID string) (*domain.Source, error)
	AddSource(ctx context.Context, source *domain.Source) error
	RemoveSource(ctx context.Context, sourceID string) error
	GetSourcesAndPollingState(ctx context.Context) ([]*domain.SourceAndPollingState, error)
	GetSourceAndPollingState(ctx context.Context, sourceID string) (*domain.SourceAndPollingState, error)
}
