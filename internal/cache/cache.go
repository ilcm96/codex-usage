package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const CacheVersion = 1

type CacheV1[T any] struct {
	Version int            `json:"version"`
	Files   map[string]T   `json:"files"`
	Meta    map[string]any `json:"meta,omitempty"`
}

func LoadCache[T any](path string) (CacheV1[T], error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return CacheV1[T]{Version: CacheVersion, Files: map[string]T{}}, nil
		}
		return CacheV1[T]{}, fmt.Errorf("failed to read cache %s: %w", path, err)
	}

	var c CacheV1[T]
	if err := json.Unmarshal(b, &c); err != nil {
		return CacheV1[T]{}, fmt.Errorf("failed to parse cache %s: %w", path, err)
	}
	if c.Version != CacheVersion || c.Files == nil {
		// Reset incompatible caches.
		return CacheV1[T]{Version: CacheVersion, Files: map[string]T{}}, nil
	}
	return c, nil
}

func SaveCache[T any](path string, value CacheV1[T]) error {
	if value.Files == nil {
		value.Files = map[string]T{}
	}
	value.Version = CacheVersion

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create cache dir %s: %w", dir, err)
	}

	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("failed to write cache tmp %s: %w", tmp, err)
	}
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("failed to encode cache %s: %w", tmp, err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("failed to close cache tmp %s: %w", tmp, err)
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("failed to replace cache %s: %w", path, err)
	}
	return nil
}
