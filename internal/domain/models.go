package domain

import "time"

type ContentType string

const (
	Tweet   ContentType = "tweet"
	Article ContentType = "article"
	Podcast ContentType = "podcast"
	Video   ContentType = "video"
	Generic ContentType = "generic"
)

type Source struct {
	ID           string      `yaml:"id"`
	DisplayName  string      `yaml:"displayName"`
	FeedURL      string      `yaml:"feedURL"`
	PollInterval string      `yaml:"pollInterval"` // e.g. "5m", "1h" – parsed later
	TypeHint     ContentType `yaml:"typeHint"`
}

type FeedItem struct {
	ID          string      `json:"id"`
	SourceID    string      `json:"source_id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Link        string      `json:"link"`
	Source      string      `json:"source"`
	PublishedAt time.Time   `json:"published"`
	Type        ContentType `json:"type"`
	Enclosure   *string     `json:"enclosure,omitempty"` // podcast audio URL
	Duration    *int        `json:"duration,omitempty"`  // seconds
	Thumbnail   *string     `json:"thumbnail,omitempty"` // video thumbnail
}

type PollingState struct {
	SourceID            string    `json:"source_id"`
	LastSuccessfulPoll  time.Time `json:"last_successful_poll"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastError           string    `json:"last_error,omitempty"`
	LastPolledAt        time.Time `json:"last_polled_at"`
}

type SourceAndPollingState struct {
	Source       Source       `json:"config"`
	PollingState PollingState `json:"state"`
}
