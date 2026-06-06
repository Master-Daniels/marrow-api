package config

import (
	"fmt"
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

type Sources struct {
	Sources []SourceDef `yaml:"sources"`
}

type SourceDef struct {
	ID           string            `yaml:"id"`
	DisplayName  string            `yaml:"display_name"`
	TypeHint     string            `yaml:"type_hint"`
	FeedURL      string            `yaml:"feed_url"`
	PollInterval string            `yaml:"poll_interval"` // e.g. "5m"
	UseRSSHub    bool              `yaml:"use_rsshub,omitempty"`
	Headers      map[string]string `yaml:"headers,omitempty"`
}

func Load(path string) (*Sources, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var sources Sources
	if err := yaml.Unmarshal(data, &sources); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &sources, nil
}

func Save(path string, sources *Sources) error {
	data, err := yaml.Marshal(sources)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func Watch(path string, onChange func()) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	hasInitialChange := false
	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					if hasInitialChange {
						onChange()
					} else {
						hasInitialChange = true
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Fatal("watch error", "error", err)
			}
		}
	}()
	watcher.Add(path)
	return nil
}
