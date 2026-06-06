package feed

import (
	"github.com/Master-Daniels/marrow/internal/config"
	"github.com/Master-Daniels/marrow/internal/domain"
)

func sourceDefToDomain(src config.SourceDef) *domain.Source {
	return &domain.Source{
		ID:           src.ID,
		DisplayName:  src.DisplayName,
		FeedURL:      src.FeedURL,
		PollInterval: src.PollInterval,
		TypeHint:     domain.ContentType(src.TypeHint),
	}
}
