package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"time"

	"github.com/Master-Daniels/marrow/internal/domain"
	"github.com/Master-Daniels/marrow/internal/errors"
	"github.com/Master-Daniels/marrow/internal/storage"
	"github.com/Master-Daniels/marrow/internal/storage/database"
	_ "modernc.org/sqlite"
)

type SQLiteRepo struct {
	db *database.Queries
}

// New creates a repo and runs migrations if needed. The path can be ":memory:" for an in-memory database or a file path for a persistent database.
func New(path string) (*SQLiteRepo, error) {
	ctx := context.Background()

	// Ensure directory exists for file-based databases
	if path != ":memory:" {
		dbDir := filepath.Dir(path)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return nil, errors.NewDatabaseError("SQLiteRepo.New", "failed to create database directory", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, errors.NewDatabaseError("SQLiteRepo.New", "failed to open database", err)
	}

	schemaPath, err := schemaFilePath()
	if err != nil {
		return nil, errors.NewDatabaseError("SQLiteRepo.New", "failed to get schema file path", err)
	}

	ddl, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, errors.NewDatabaseError("SQLiteRepo.New", "failed to read schema file", err)
	}

	// create tables
	if _, err := db.ExecContext(ctx, string(ddl)); err != nil {
		return nil, errors.NewDatabaseError("SQLiteRepo.New", "failed to execute schema", err)
	}

	sqlcdb := database.New(db)

	return &SQLiteRepo{db: sqlcdb}, nil
}

// InsertOrReplace implements the Repository interface
func (r *SQLiteRepo) Insert(ctx context.Context, item *domain.FeedItem) error {
	params := database.InsertOrReplaceItemParams{
		ID:          item.ID,
		SourceID:    nullableString(item.SourceID),
		Title:       nullableString(item.Title),
		Description: nullableString(item.Description),
		Link:        nullableString(item.Link),
		Source:      nullableString(item.Source),
		PublishedAt: nullableString(item.PublishedAt.Format(time.RFC3339)),
		Type:        nullableString(string(item.Type)),
		Enclosure:   nullableStringPtr(item.Enclosure),
		Duration:    nullableInt64Ptr(item.Duration),
		Thumbnail:   nullableStringPtr(item.Thumbnail),
	}

	_, err := r.db.InsertOrReplaceItem(ctx, params)
	return err
}

// All implements the Repository interface
func (r *SQLiteRepo) All(ctx context.Context, params storage.PaginationParams) ([]*domain.FeedItem, error) {
	var dbItems []database.Items
	var err error

	// Handle cursor-based pagination
	if params.Cursor != "" {
		// cursor, err := decodeCursor(params.Cursor)
		// if err != nil {
		// 	return nil, errors.NewDatabaseError("SQLiteRepo.All", "invalid cursor", err)
		// }

		// dbItems, err = r.db.GetAllItemsWithCursor(ctx, database.GetAllItemsWithCursorParams{
		// 	PublishedAt: cursor.PublishedAt,
		// 	ID:          cursor.ID,
		// 	Limit:       int32(params.Limit),
		// })
	} else {
		// Handle offset-based pagination
		dbItems, err = r.db.GetAllItems(ctx, database.GetAllItemsParams{
			Limit:  int64(params.Limit),
			Offset: int64(params.Offset),
		})
	}

	if err != nil {
		return nil, err
	}

	return convertToDomainItems(dbItems), nil
}

// GetBySource implements the Repository interface
func (r *SQLiteRepo) GetBySource(ctx context.Context, sourceID string, params storage.PaginationParams) ([]*domain.FeedItem, error) {
	var dbItems []database.Items
	var err error

	// Handle cursor-based pagination
	if params.Cursor != "" {
		// cursor, err := decodeCursor(params.Cursor)
		// if err != nil {
		// 	return nil, errors.NewDatabaseError("SQLiteRepo.GetBySource", "invalid cursor", err)
		// }

		// dbItems, err = r.db.GetItemsBySourceWithCursor(ctx, database.GetItemsBySourceWithCursorParams{
		// 	SourceID:    sourceID,
		// 	PublishedAt: cursor.PublishedAt,
		// 	ID:          cursor.ID,
		// 	Limit:       int32(params.Limit),
		// })
	} else {
		// Handle offset-based pagination
		dbItems, err = r.db.GetItemsBySource(ctx, database.GetItemsBySourceParams{
			SourceID: nullableString(sourceID),
			Limit:    int64(params.Limit),
			Offset:   int64(params.Offset),
		})
	}

	if err != nil {
		return nil, err
	}

	return convertToDomainItems(dbItems), nil
}

// GetAfter implements the Repository interface
func (r *SQLiteRepo) GetAfter(ctx context.Context, after time.Time, params storage.PaginationParams) ([]*domain.FeedItem, error) {
	var dbItems []database.Items
	var err error

	// Handle cursor-based pagination
	if params.Cursor != "" {
		// cursor, err := decodeCursor(params.Cursor)
		// if err != nil {
		// 	return nil, errors.NewDatabaseError("SQLiteRepo.GetAfter", "invalid cursor", err)
		// }

		// dbItems, err = r.db.GetItemsAfterWithCursor(ctx, database.GetItemsAfterWithCursorParams{
		// 	After:       after.Format(time.RFC3339),
		// 	PublishedAt: cursor.PublishedAt,
		// 	ID:          cursor.ID,
		// 	Limit:       int32(params.Limit),
		// })
	} else {
		// Handle offset-based pagination
		dbItems, err = r.db.GetItemsAfter(ctx, database.GetItemsAfterParams{
			PublishedAt: nullableString(after.Format(time.RFC3339)),
			Limit:       int64(params.Limit),
			Offset:      int64(params.Offset),
		})
	}

	if err != nil {
		return nil, err
	}

	return convertToDomainItems(dbItems), nil
}

// GetByType implements the Repository interface
func (r *SQLiteRepo) GetByType(ctx context.Context, contentType domain.ContentType, params storage.PaginationParams) ([]*domain.FeedItem, error) {
	var dbItems []database.Items
	var err error

	// Handle cursor-based pagination
	if params.Cursor != "" {
		// cursor, err := decodeCursor(params.Cursor)
		// if err != nil {
		// 	return nil, errors.NewDatabaseError("SQLiteRepo.GetByType", "invalid cursor", err)
		// }

		// dbItems, err = r.db.GetItemsByTypeWithCursor(ctx, database.GetItemsByTypeWithCursorParams{
		// 	Type:        string(contentType),
		// 	PublishedAt: cursor.PublishedAt,
		// 	ID:          cursor.ID,
		// 	Limit:       int32(params.Limit),
		// })
	} else {
		// Handle offset-based pagination
		dbItems, err = r.db.GetItemsByType(ctx, database.GetItemsByTypeParams{
			Type:   nullableString(string(contentType)),
			Limit:  int64(params.Limit),
			Offset: int64(params.Offset),
		})
	}

	if err != nil {
		return nil, err
	}

	return convertToDomainItems(dbItems), nil
}

// GetBySourceAndType implements the Repository interface
func (r *SQLiteRepo) GetBySourceAndType(ctx context.Context, sourceID string, contentType domain.ContentType, params storage.PaginationParams) ([]*domain.FeedItem, error) {
	var dbItems []database.Items
	var err error

	// Handle cursor-based pagination
	if params.Cursor != "" {
		// cursor, err := decodeCursor(params.Cursor)
		// if err != nil {
		// 	return nil, errors.NewDatabaseError("SQLiteRepo.GetBySourceAndType", "invalid cursor", err)
		// }

		// dbItems, err = r.db.GetItemsBySourceAndTypeWithCursor(ctx, database.GetItemsBySourceAndTypeWithCursorParams{
		// 	SourceID:    sourceID,
		// 	Type:        string(contentType),
		// 	PublishedAt: cursor.PublishedAt,
		// 	ID:          cursor.ID,
		// 	Limit:       int32(params.Limit),
		// })
	} else {
		// Handle offset-based pagination
		dbItems, err = r.db.GetItemsBySourceAndType(ctx, database.GetItemsBySourceAndTypeParams{
			SourceID: nullableString(sourceID),
			Type:     nullableString(string(contentType)),
			Limit:    int64(params.Limit),
			Offset:   int64(params.Offset),
		})
	}

	if err != nil {
		return nil, err
	}

	return convertToDomainItems(dbItems), nil
}

// GetAfterAndType implements the Repository interface
func (r *SQLiteRepo) GetAfterAndType(ctx context.Context, after time.Time, contentType domain.ContentType, params storage.PaginationParams) ([]*domain.FeedItem, error) {
	var dbItems []database.Items
	var err error

	// Handle cursor-based pagination
	if params.Cursor != "" {
		// cursor, err := decodeCursor(params.Cursor)
		// if err != nil {
		// 	return nil, errors.NewDatabaseError("SQLiteRepo.GetAfterAndType", "invalid cursor", err)
		// }

		// dbItems, err = r.db.GetItemsAfterAndTypeWithCursor(ctx, database.GetItemsAfterAndTypeWithCursorParams{
		// 	After:       after.Format(time.RFC3339),
		// 	Type:        string(contentType),
		// 	PublishedAt: cursor.PublishedAt,
		// 	ID:          cursor.ID,
		// 	Limit:       int64(params.Limit),
		// })
	} else {
		// Handle offset-based pagination
		dbItems, err = r.db.GetItemsAfterAndType(ctx, database.GetItemsAfterAndTypeParams{
			PublishedAt: nullableString(after.Format(time.RFC3339)),
			Type:        nullableString(string(contentType)),
			Limit:       int64(params.Limit),
			Offset:      int64(params.Offset),
		})
	}

	if err != nil {
		return nil, err
	}

	return convertToDomainItems(dbItems), nil
}

// GetAfterAndSource implements the Repository interface
func (r *SQLiteRepo) GetAfterAndSource(ctx context.Context, after time.Time, sourceID string, params storage.PaginationParams) ([]*domain.FeedItem, error) {
	var dbItems []database.Items
	var err error

	// Handle cursor-based pagination
	if params.Cursor != "" {
		// cursor, err := decodeCursor(params.Cursor)
		// if err != nil {
		// 	return nil, errors.NewDatabaseError("SQLiteRepo.GetAfterAndSource", "invalid cursor", err)
		// }

		// dbItems, err = r.db.GetItemsAfterAndSourceWithCursor(ctx, database.GetItemsAfterAndSourceWithCursorParams{
		// 	After:       after.Format(time.RFC3339),
		// 	SourceID:    sourceID,
		// 	PublishedAt: cursor.PublishedAt,
		// 	ID:          cursor.ID,
		// 	Limit:       int64(params.Limit),
		// })
	} else {
		// Handle offset-based pagination
		dbItems, err = r.db.GetItemsAfterAndSource(ctx, database.GetItemsAfterAndSourceParams{
			PublishedAt: nullableString(after.Format(time.RFC3339)),
			SourceID:    nullableString(sourceID),
			Limit:       int64(params.Limit),
			Offset:      int64(params.Offset),
		})
	}

	if err != nil {
		return nil, err
	}

	return convertToDomainItems(dbItems), nil
}

// Latest implements the Repository interface
func (r *SQLiteRepo) Latest(ctx context.Context, limit int) ([]*domain.FeedItem, error) {
	dbItems, err := r.db.GetLatestItems(ctx, int64(limit))
	if err != nil {
		return nil, err
	}

	return convertToDomainItems(dbItems), nil
}

// Cleanup implements the Repository interface
func (r *SQLiteRepo) Cleanup(ctx context.Context) error {
	return r.db.CleanupOldItems(ctx)
}

// SaveOrUpdatePollingState implements storage.Repository.
func (r *SQLiteRepo) SaveOrUpdatePollingState(ctx context.Context, state *domain.PollingState) error {
	databaseState := database.UpsertPollingStateParams{
		SourceID:            state.SourceID,
		LastSuccessfulPoll:  nullableString(state.LastSuccessfulPoll.Format(time.RFC3339)),
		ConsecutiveFailures: nullableInt64Ptr(&state.ConsecutiveFailures),
		LastError:           nullableString(state.LastError),
		LastPolledAt:        nullableString(state.LastPolledAt.Format(time.RFC3339)),
	}
	return r.db.UpsertPollingState(ctx, databaseState)
}

// GetPollingState implements storage.Repository.
func (r *SQLiteRepo) GetPollingState(ctx context.Context, sourceID string) (*domain.PollingState, error) {
	var pollingState database.PollingState
	var err error

	if sourceID == "" {
		return nil, errors.NewDatabaseError("SQLite.GetPollingSate", "sourceID is required", nil)
	}
	pollingState, err = r.db.GetPollingState(ctx, sourceID)

	if err != nil {
		return nil, err
	}
	return convertToDomainPollingState(&pollingState), nil
}

// GetAllSources implements storage.Repository.
func (r *SQLiteRepo) GetAllSources(ctx context.Context) ([]*domain.Source, error) {
	dbSources, err := r.db.GetAllSources(ctx)
	if err != nil {
		return nil, err
	}

	sources := make([]*domain.Source, 0, len(dbSources))
	for _, dbSource := range dbSources {
		sources = append(sources, convertToDomainSource(dbSource))
	}

	return sources, nil
}

// GetSource implements storage.Repository.
func (r *SQLiteRepo) GetSource(ctx context.Context, sourceID string) (*domain.Source, error) {
	if sourceID == "" {
		return nil, errors.NewDatabaseError("SQLiteRepo.GetSource", "sourceID is required", nil)
	}

	dbSource, err := r.db.GetSource(ctx, sourceID)
	if err != nil {
		return nil, err
	}

	return convertToDomainSource(dbSource), nil
}

// AddSource implements storage.Repository.
func (r *SQLiteRepo) AddSource(ctx context.Context, source *domain.Source) error {
	if source == nil {
		return errors.NewDatabaseError("SQLiteRepo.AddSource", "source is required", nil)
	}

	return r.db.UpsertSource(ctx, database.UpsertSourceParams{
		ID:           source.ID,
		DisplayName:  nullableString(source.DisplayName),
		FeedUrl:      nullableString(source.FeedURL),
		PollInterval: nullableString(source.PollInterval),
		TypeHint:     nullableString(string(source.TypeHint)),
	})
}

// RemoveSource implements storage.Repository.
func (r *SQLiteRepo) RemoveSource(ctx context.Context, sourceID string) error {
	if sourceID == "" {
		return errors.NewDatabaseError("SQLiteRepo.RemoveSource", "sourceID is required", nil)
	}

	if err := r.db.DeletePollingStateBySourceID(ctx, sourceID); err != nil {
		return errors.NewDatabaseError("SQLiteRepo.RemoveSource", "failed to delete polling state", err)
	}
	return r.db.DeleteSource(ctx, sourceID)
}

// GetSourcesAndPollingState implements storage.Repository.
func (r *SQLiteRepo) GetSourcesAndPollingState(ctx context.Context) ([]*domain.SourceAndPollingState, error) {
	dbRows, err := r.db.GetSourcesAndPollingState(ctx, nil)
	if err != nil {
		return nil, err
	}

	results := make([]*domain.SourceAndPollingState, 0, len(dbRows))
	for _, row := range dbRows {
		results = append(results, convertToDomainSourceAndPollingState(row))
	}

	return results, nil
}

// GetSourceAndPollingState implements storage.Repository.
func (r *SQLiteRepo) GetSourceAndPollingState(ctx context.Context, sourceID string) (*domain.SourceAndPollingState, error) {
	if sourceID == "" {
		return nil, errors.NewDatabaseError("SQLiteRepo.GetSourceAndPollingState", "sourceID is required", nil)
	}

	row, err := r.db.GetSourcesAndPollingState(ctx, sourceID)
	if err != nil {
		return nil, err
	}

	result := convertToDomainSourceAndPollingState(row[0])
	return result, nil
}
