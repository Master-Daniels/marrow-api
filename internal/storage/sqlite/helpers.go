package sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Master-Daniels/marrow/internal/domain"
	"github.com/Master-Daniels/marrow/internal/storage/database"
)

func schemaFilePath() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("unable to determine schema file path")
	}

	return filepath.Join(filepath.Dir(filename), "schema.sql"), nil
}

// nullableValue converts a pointer-based domain field into its SQL nullable type.
// This is a generic helper for any SQL null wrapper that can be created from a concrete value.
func nullableValue[T any, N any](value *T, convert func(T) N) N {
	var zero N
	if value == nil {
		return zero
	}
	return convert(*value)
}

func nullableString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func nullableStringPtr(value *string) sql.NullString {
	return nullableValue(value, func(v string) sql.NullString {
		return sql.NullString{String: v, Valid: true}
	})
}

func nullableInt64Ptr(value *int) sql.NullInt64 {
	return nullableValue(value, func(v int) sql.NullInt64 {
		return sql.NullInt64{Int64: int64(v), Valid: true}
	})
}

// Helper function to convert database items to domain items
func convertToDomainItems(dbItems []database.Items) []*domain.FeedItem {
	items := make([]*domain.FeedItem, len(dbItems))
	for i, dbItem := range dbItems {
		publishedAt, _ := time.Parse(time.RFC3339, dbItem.PublishedAt.String)

		var enclosure, thumbnail *string
		if dbItem.Enclosure.Valid {
			enclosure = &dbItem.Enclosure.String
		}
		if dbItem.Thumbnail.Valid {
			thumbnail = &dbItem.Thumbnail.String
		}

		var duration *int
		if dbItem.Duration.Valid {
			d := int(dbItem.Duration.Int64)
			duration = &d
		}

		items[i] = &domain.FeedItem{
			ID:          dbItem.ID,
			SourceID:    dbItem.SourceID.String,
			Title:       dbItem.Title.String,
			Description: dbItem.Description.String,
			Link:        dbItem.Link.String,
			Source:      dbItem.Source.String,
			PublishedAt: publishedAt,
			Type:        domain.ContentType(dbItem.Type.String),
			Enclosure:   enclosure,
			Duration:    duration,
			Thumbnail:   thumbnail,
		}
	}
	return items
}

func convertToDomainPollingState(dbState *database.PollingState) *domain.PollingState {
	var lastSuccessfulPoll time.Time
	if dbState.LastSuccessfulPoll.Valid {
		lastSuccessfulPoll, _ = time.Parse(time.RFC3339, dbState.LastSuccessfulPoll.String)
	}

	var lastPolledAt time.Time
	if dbState.LastPolledAt.Valid {
		lastPolledAt, _ = time.Parse(time.RFC3339, dbState.LastPolledAt.String)
	}

	lastError := ""
	if dbState.LastError.Valid {
		lastError = dbState.LastError.String
	}

	return &domain.PollingState{
		SourceID:            dbState.SourceID,
		LastSuccessfulPoll:  lastSuccessfulPoll,
		ConsecutiveFailures: int(dbState.ConsecutiveFailures.Int64),
		LastError:           lastError,
		LastPolledAt:        lastPolledAt,
	}
}

func convertToDomainSource(dbSource database.Sources) *domain.Source {
	return &domain.Source{
		ID:           dbSource.ID,
		DisplayName:  dbSource.DisplayName.String,
		FeedURL:      dbSource.FeedUrl.String,
		PollInterval: dbSource.PollInterval.String,
		TypeHint:     domain.ContentType(dbSource.TypeHint.String),
	}
}

func convertToDomainSourceAndPollingState(row database.GetSourcesAndPollingStateRow) *domain.SourceAndPollingState {
	var lastSuccessfulPoll time.Time
	if row.LastSuccessfulPoll.Valid {
		lastSuccessfulPoll, _ = time.Parse(time.RFC3339, row.LastSuccessfulPoll.String)
	}

	var lastPolledAt time.Time
	if row.LastPolledAt.Valid {
		lastPolledAt, _ = time.Parse(time.RFC3339, row.LastPolledAt.String)
	}

	lastError := ""
	if row.LastError.Valid {
		lastError = row.LastError.String
	}

	return &domain.SourceAndPollingState{
		Source: domain.Source{
			ID:           row.ID,
			DisplayName:  row.DisplayName.String,
			FeedURL:      row.FeedUrl.String,
			PollInterval: row.PollInterval.String,
			TypeHint:     domain.ContentType(row.TypeHint.String),
		},
		PollingState: domain.PollingState{
			SourceID:            row.ID,
			LastSuccessfulPoll:  lastSuccessfulPoll,
			ConsecutiveFailures: int(row.ConsecutiveFailures.Int64),
			LastError:           lastError,
			LastPolledAt:        lastPolledAt,
		},
	}
}

// Cursor represents a pagination cursor
type Cursor struct {
	PublishedAt string `json:"published_at"`
	ID          string `json:"id"`
}

// decodeCursor decodes a cursor string into a Cursor struct
func decodeCursor(cursor string) (*Cursor, error) {
	var c Cursor
	err := json.Unmarshal([]byte(cursor), &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// encodeCursor encodes a Cursor struct into a cursor string
func encodeCursor(c *Cursor) (string, error) {
	bytes, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
