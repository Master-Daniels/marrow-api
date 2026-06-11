-- Drop existing tables to recreate with correct schema
DROP TABLE IF EXISTS items;

DROP TABLE IF EXISTS sources;

-- Create sources table matching the Source domain model
CREATE TABLE sources (
    id TEXT PRIMARY KEY,
    display_name TEXT,
    feed_url TEXT,
    poll_interval TEXT,
    type_hint TEXT
);

-- Create items table matching the FeedItem domain model
CREATE TABLE items (
    id TEXT PRIMARY KEY,
    source_id TEXT,
    title TEXT,
    author TEXT,
    description TEXT,
    link TEXT,
    source TEXT,
    published_at TEXT,
    type TEXT,
    enclosure TEXT,
    duration INTEGER,
    thumbnail TEXT
);

-- Create polling_state table to track polling metadata for each source
CREATE TABLE IF NOT EXISTS polling_state (
    source_id TEXT PRIMARY KEY,
    last_successful_poll TEXT,
    consecutive_failures INTEGER,
    last_error TEXT,
    last_polled_at TEXT
);

-- Composite index for GetBySourceAndType queries
CREATE INDEX IF NOT EXISTS idx_items_source_type ON items (source_id, type);

-- Composite index for GetAfterAndType queries with cursor-based pagination support
CREATE INDEX IF NOT EXISTS idx_items_published_type ON items (published_at DESC, type, id DESC);

-- Composite index for GetAfterAndSource queries with cursor-based pagination support
CREATE INDEX IF NOT EXISTS idx_items_published_source ON items (
    published_at DESC,
    source_id,
    id DESC
);

-- Composite index for Latest queries and cursor-based pagination
CREATE INDEX IF NOT EXISTS idx_items_published_desc ON items (published_at DESC, id DESC);

-- Composite index for source_id with published_at and id for cursor-based pagination
CREATE INDEX IF NOT EXISTS idx_items_source_published ON items (
    source_id,
    published_at DESC,
    id DESC
);

-- Composite index for type with published_at and id for cursor-based pagination
CREATE INDEX IF NOT EXISTS idx_items_type_published ON items (type, published_at DESC, id DESC);