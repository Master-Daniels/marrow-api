package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

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

func ResolveConfigPath(configPath string) (string, error) {
	if configPath == "" {
		return "", errors.New("Config file not found, stopping the application, please provide a config file path")
	}

	if info, err := os.Stat(configPath); err == nil && info.IsDir() {
		return filepath.Join(configPath, "sources.yml"), nil
	}

	return configPath, nil
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

	absPath, err := filepath.Abs(path)
	if err != nil {
		watcher.Close()
		return fmt.Errorf("failed to resolve config path: %w", err)
	}

	dir := filepath.Dir(absPath)
	fileName := filepath.Base(absPath)

	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return fmt.Errorf("failed to watch config directory %q: %w", dir, err)
	}

	var lastModTime time.Time
	var lastSize int64
	if info, err := os.Stat(absPath); err == nil {
		lastModTime = info.ModTime()
		lastSize = info.Size()
	}

	go func() {
		defer watcher.Close()

		var debounceTimer *time.Timer

		triggerReload := func() {
			if debounceTimer != nil {
				debounceTimer.Stop()
			}

			debounceTimer = time.AfterFunc(300*time.Millisecond, onChange)
		}

		pollTicker := time.NewTicker(2 * time.Second)
		defer pollTicker.Stop()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if filepath.Base(event.Name) != fileName {
					continue
				}

				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove|fsnotify.Chmod) != 0 {
					triggerReload()
				}

			case <-pollTicker.C:
				info, err := os.Stat(absPath)
				if err != nil {
					continue
				}

				if !info.ModTime().Equal(lastModTime) || info.Size() != lastSize {
					lastModTime = info.ModTime()
					lastSize = info.Size()
					triggerReload()
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}

				log.Printf("config watch error: %v", err)
			}
		}
	}()

	return nil
}