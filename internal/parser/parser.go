package parser

import (
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/Master-Daniels/marrow/internal/config"
	"github.com/Master-Daniels/marrow/internal/domain"
	"github.com/mmcdole/gofeed"
)

// Parser knows how to turn a raw HTTP body into []FeedItem.
// It can have extra helper methods if you support multiple source types.
type Parser struct {
	fp *gofeed.Parser
}

// New returns a ready‑to‑use parser.
func New() *Parser {
	return &Parser{
		fp: gofeed.NewParser(),
	}
}

// Parse reads the body, parses the feed, and returns domain items.
// The returned slice is *already* filtered for non‑empty GUID/link.
func (p *Parser) Parse(body io.Reader, src config.SourceDef) ([]*domain.FeedItem, error) {
	// gofeed works with XML, but it also accepts JSON for some feeds.
	feed, err := p.fp.Parse(body)
	if err != nil {
		return nil, err
	}

	var out []*domain.FeedItem
	for _, entry := range feed.Items {
		it := mapEntry(entry, src)
		if it != nil {
			out = append(out, it)
		}
	}
	return out, nil
}

// mapEntry converts a gofeed.Item into our domain.FeedItem.
// It also applies some source‑type hints (e.g., "tweet", "podcast") if the
// config provides a `type_hint`.
func mapEntry(e *gofeed.Item, src config.SourceDef) *domain.FeedItem {
	// Guard against completely empty items.
	if e.GUID == "" && e.Link == "" {
		return nil
	}

	// Parse publication date – gofeed already does most of the heavy lifting.
	var pub time.Time
	if e.PublishedParsed != nil {
		pub = *e.PublishedParsed
	} else if e.UpdatedParsed != nil {
		pub = *e.UpdatedParsed
	} else {
		pub = time.Now()
	}

	// Determine content type based on source hint or auto-detect
	contentType := domain.Generic
	if src.TypeHint != "" {
		contentType = domain.ContentType(src.TypeHint)
	} else {
		// Auto-detect content type based on feed characteristics
		if len(e.Enclosures) > 0 {
			enc := e.Enclosures[0]
			if enc.Type != "" {
				if enc.Type == "audio/mpeg" || enc.Type == "audio/mp3" {
					contentType = domain.Podcast
				} else if enc.Type == "video/mp4" || enc.Type == "video/mpeg" {
					contentType = domain.Video
				}
			}
		} else if src.FeedURL != "" && (src.FeedURL == "https://nitter.net" || src.FeedURL == "https://twitter.com") {
			contentType = domain.Tweet
		} else {
			contentType = domain.Article
		}
	}

	// Enclosure handling – podcast/audio tracks are usually in the first enclosure.
	var enclosureURL *string
	var duration *int
	if len(e.Enclosures) > 0 {
		enc := e.Enclosures[0]
		if enc.URL != "" {
			enclosureURL = &enc.URL
		}
		if enc.Length != "" {
			// Length field is sometimes used for duration (seconds) – try to parse.
			if d, err := time.ParseDuration(enc.Length + "s"); err == nil {
				sec := int(d.Seconds())
				duration = &sec
			}
		}
	}

	// Thumbnail – many RSS feeds expose an image tag in the entry.
	var thumb *string
	if e.Image != nil && e.Image.URL != "" {
		thumb = &e.Image.URL
	}

	// Special handling for tweets
	if contentType == domain.Tweet {
		if e.Description != "" {
			// For tweets, use the description as the content
			// Clean up any HTML if needed
		}
	}

	// Special handling for podcasts
	if contentType == domain.Podcast {
		if e.ITunesExt != nil && e.ITunesExt.Duration != "" {
			// Parse iTunes duration format (HH:MM:SS or MM:SS or SS)
			parts := strings.Split(e.ITunesExt.Duration, ":")
			var totalSeconds int
			switch len(parts) {
			case 3:
				hours, _ := strconv.Atoi(parts[0])
				minutes, _ := strconv.Atoi(parts[1])
				seconds, _ := strconv.Atoi(parts[2])
				totalSeconds = hours*3600 + minutes*60 + seconds
			case 2:
				minutes, _ := strconv.Atoi(parts[0])
				seconds, _ := strconv.Atoi(parts[1])
				totalSeconds = minutes*60 + seconds
			case 1:
				totalSeconds, _ = strconv.Atoi(parts[0])
			}
			if totalSeconds > 0 {
				duration = &totalSeconds
			}
		}
		if e.ITunesExt != nil && e.ITunesExt.Image != "" {
			thumb = &e.ITunesExt.Image
		}
	}

	// Special handling for videos
	if contentType == domain.Video && e.ITunesExt != nil && e.ITunesExt.Image != "" {
		thumb = &e.ITunesExt.Image
	}

	return &domain.FeedItem{
		Title:       e.Title,
		Description: e.Description,
		Link:        e.Link,
		// Author:      e.Author.Name, // TODO: support multiple authors
		PublishedAt: pub,
		Enclosure:   enclosureURL,
		Duration:    duration,
		Thumbnail:   thumb,
		Type:        contentType,
		// The GUID field from the feed (if present) is not stored directly.
		// The poller will generate a deterministic ID from source+link.
	}
}
