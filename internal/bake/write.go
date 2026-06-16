package bake

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// assetDoc is the on-disk JSON shape: the scene as a motion-tools snapshot
// (verbatim, so the web side parses it with SnapshotProto.fromJson) plus the
// playback track.
type assetDoc struct {
	Scene json.RawMessage `json:"scene"`
	Track []TrackStep     `json:"track"`
}

// WriteAsset serializes an Asset to path as {"scene": <snapshot>, "track": [...]},
// creating parent directories as needed.
func WriteAsset(path string, a *Asset) error {
	sceneJSON, err := a.Scene.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal scene: %w", err)
	}
	doc := assetDoc{Scene: sceneJSON, Track: a.Track}
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal asset: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir for %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
