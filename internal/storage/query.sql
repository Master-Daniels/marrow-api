-- Insert or update a feed item (handle duplicates)
-- name: InsertOrReplaceItem :one
INSERT
    OR
REPLACE INTO
    items (
        id,
        source_id,
        title,
        description,
        link,
        source,
        published_at,
        type,
        enclosure,
        duration,
        thumbnail
    )
VALUES (
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?
    ) RETURNING *;

-- Upsert polling state for a source
-- name: UpsertPollingState :exec
INSERT INTO
    polling_state (
        source_id,
        last_successful_poll,
        consecutive_failures,
        last_error,
        last_polled_at
    )
VALUES (?, ?, ?, ?, ?) ON CONFLICT (source_id) DO
UPDATE
SET
    last_successful_poll = excluded.last_successful_poll,
    consecutive_failures = excluded.consecutive_failures,
    last_error = excluded.last_error,
    last_polled_at = excluded.last_polled_at;

-- Upsert source metadata
-- name: UpsertSource :exec
INSERT
    OR
REPLACE INTO
    sources (
        id,
        display_name,
        feed_url,
        poll_interval,
        type_hint
    )
VALUES (?, ?, ?, ?, ?);

-- Get all sources
-- name: GetAllSources :many
SELECT
    id,
    display_name,
    feed_url,
    poll_interval,
    type_hint
FROM sources
ORDER BY id;

-- Get source by ID
-- name: GetSource :one
SELECT
    id,
    display_name,
    feed_url,
    poll_interval,
    type_hint
FROM sources
WHERE
    id = ?;

-- Get sources along with their polling state (pass NULL to get all, or an ID to get one)
-- name: GetSourcesAndPollingState :many
SELECT s.id, s.display_name, s.feed_url, s.poll_interval, s.type_hint, p.last_successful_poll, p.consecutive_failures, p.last_error, p.last_polled_at
FROM sources s
    LEFT JOIN polling_state p ON s.id = p.source_id
WHERE (
        sqlc.narg ('id') IS NULL
        OR s.id = sqlc.narg ('id')
    )
ORDER BY s.id;

-- Update source metadata
-- name: UpdateSource :exec
UPDATE sources
SET
    display_name = ?,
    feed_url = ?,
    poll_interval = ?,
    type_hint = ?
WHERE
    id = ?;

-- Delete source by ID
-- name: DeleteSource :exec
DELETE FROM sources WHERE id = ?;

-- Delete polling state for a removed source
-- name: DeletePollingStateBySourceID :exec
DELETE FROM polling_state WHERE source_id = ?;

-- Get the polling state for a source
-- name: GetPollingState :one
SELECT
    source_id,
    last_successful_poll,
    consecutive_failures,
    last_error,
    last_polled_at
FROM polling_state
WHERE
    source_id = ?;

-- insert many items with a single query
-- name: InsertOrReplaceItems :many
INSERT
    OR
REPLACE INTO
    items (
        id,
        source_id,
        title,
        description,
        link,
        source,
        published_at,
        type,
        enclosure,
        duration,
        thumbnail
    )
VALUES (
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?
    ) RETURNING *;
-- Get all items with pagination
-- name: GetAllItems :many
SELECT * FROM items ORDER BY published_at DESC LIMIT ? OFFSET ?;

-- Get items by source ID with pagination
-- name: GetItemsBySource :many
SELECT *
FROM items
WHERE
    source_id = ?
ORDER BY published_at DESC
LIMIT ?
OFFSET
    ?;

-- Get items published after a certain time with pagination
-- name: GetItemsAfter :many
SELECT *
FROM items
WHERE
    published_at > ?
ORDER BY published_at DESC
LIMIT ?
OFFSET
    ?;

-- Get items by type with pagination
-- name: GetItemsByType :many
SELECT *
FROM items
WHERE
    type = ?
ORDER BY published_at DESC
LIMIT ?
OFFSET
    ?;

-- Get items by source ID and type with pagination
-- name: GetItemsBySourceAndType :many
SELECT *
FROM items
WHERE
    source_id = ?
    AND type = ?
ORDER BY published_at DESC
LIMIT ?
OFFSET
    ?;

-- Get items published after a certain time and by type with pagination
-- name: GetItemsAfterAndType :many
SELECT *
FROM items
WHERE
    published_at > ?
    AND type = ?
ORDER BY published_at DESC
LIMIT ?
OFFSET
    ?;

-- Get items published after a certain time and by source with pagination
-- name: GetItemsAfterAndSource :many
SELECT *
FROM items
WHERE
    published_at > ?
    AND source_id = ?
ORDER BY published_at DESC
LIMIT ?
OFFSET
    ?;

-- Get the latest N items ordered by published_at desc
-- name: GetLatestItems :many
SELECT * FROM items ORDER BY published_at DESC LIMIT ?;

-- Cleanup old items
-- name: CleanupOldItems :exec
DELETE FROM items WHERE published_at < datetime('now', '-30 days');