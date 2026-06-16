// Command bake plans Beanjamin's brew sequence for each arm with real Viam
// motion planning and writes the static replay assets the web app consumes:
// web/static/trajectories/<arm>.brew.json (scene snapshot + per-step pose track).
//
// Run from the repo root:
//
//	go run ./cmd/bake
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"go.viam.com/rdk/logging"

	"homepage-simulated-arm-demo/internal/bake"
)

// arms is the lineup the demo toggles between. Both plan the same brew sequence.
var arms = []string{"xarm6", "ur5e"}

func main() {
	ctx := context.Background()
	logger := logging.NewLogger("bake")

	baker := bake.Baker{
		KinematicsDir: "data/kinematics",
		ConfigPath:    "data/beanjamin-config.merged.json",
	}
	outDir := filepath.Join("web", "static", "trajectories")

	for _, arm := range arms {
		asset, err := baker.Build(ctx, logger, arm)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bake %s: %v\n", arm, err)
			os.Exit(1)
		}
		out := filepath.Join(outDir, arm+".brew.json")
		if err := bake.WriteAsset(out, asset); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", arm, err)
			os.Exit(1)
		}
		fmt.Printf("✓ %s: %d scene transforms, %d track steps -> %s\n",
			arm, len(asset.Scene.Transforms()), len(asset.Track), out)
	}
}
