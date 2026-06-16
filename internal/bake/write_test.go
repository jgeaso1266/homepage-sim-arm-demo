package bake

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.viam.com/rdk/logging"
	"go.viam.com/test"
	"google.golang.org/protobuf/encoding/protojson"

	drawv1 "github.com/viam-labs/motion-tools/draw/v1"
)

func TestWriteAsset_roundTrip(t *testing.T) {
	a, err := testBaker().Build(context.Background(), logging.NewTestLogger(t), "xarm6")
	test.That(t, err, test.ShouldBeNil)

	path := filepath.Join(t.TempDir(), "xarm6.brew.json")
	test.That(t, WriteAsset(path, a), test.ShouldBeNil)

	raw, err := os.ReadFile(path)
	test.That(t, err, test.ShouldBeNil)

	var doc struct {
		Scene json.RawMessage `json:"scene"`
		Track []TrackStep     `json:"track"`
	}
	test.That(t, json.Unmarshal(raw, &doc), test.ShouldBeNil)

	// Track survives the round trip.
	test.That(t, len(doc.Track), test.ShouldEqual, len(a.Track))
	test.That(t, len(doc.Track[0].Poses), test.ShouldEqual, len(a.Track[0].Poses))

	// Scene re-parses as a Snapshot proto (the same protojson the web side reads)
	// with the same transform count.
	var reloaded drawv1.Snapshot
	test.That(t, protojson.Unmarshal(doc.Scene, &reloaded), test.ShouldBeNil)
	test.That(t, len(reloaded.GetTransforms()), test.ShouldEqual, len(a.Scene.Transforms()))
}
