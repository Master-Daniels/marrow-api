package poller

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// DeduplicateItem represents an item to be deduplicated
type DeduplicateItem struct {
	GUID        string
	Link        string
	Title       string
	PubDate     time.Time
	Source      string
	Description string
	LastUpdated time.Time
}

// DeduplicateFunction deduplicates an item based on the recommended approaches
func deduplicateFunction(item DeduplicateItem) (string, error) {
	if item.GUID != "" {
		return item.GUID, nil
	}

	if item.Link != "" {
		hash := sha256.Sum256([]byte(item.Link))
		return hex.EncodeToString(hash[:]), nil
	}

	// If Link is not available, use composite hashing
	// Use a combination of Title, PubDate, and Source
	compositeKey := fmt.Sprintf("%s%s%s", strings.TrimSpace(item.Title), item.PubDate.Format(time.RFC3339), item.Source)
	hash := sha256.Sum256([]byte(compositeKey))
	return hex.EncodeToString(hash[:]), nil
}
