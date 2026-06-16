package bake

import (
	"context"
	"strings"
	"testing"

	"go.viam.com/rdk/logging"
	"go.viam.com/test"
)

func testBaker() Baker {
	return Baker{
		KinematicsDir: "../../data/kinematics",
		ConfigPath:    "../../data/beanjamin-config.merged.json",
	}
}

func TestBuildAsset(t *testing.T) {
	a, err := testBaker().Build(context.Background(), logging.NewTestLogger(t), "xarm6")
	test.That(t, err, test.ShouldBeNil)

	// Scene carries the obstacle + arm/tool geometries at the start pose.
	test.That(t, len(a.Scene.Transforms()), test.ShouldBeGreaterThan, 0)

	// Track has a step per planned trajectory entry.
	test.That(t, len(a.Track), test.ShouldBeGreaterThan, 0)

	// Every step poses the same set of moving frames, and only moving frames.
	first := a.Track[0].Poses
	test.That(t, len(first), test.ShouldBeGreaterThan, 0)
	for name := range first {
		test.That(t, isMoving(name), test.ShouldBeTrue)
	}
	test.That(t, len(a.Track[len(a.Track)-1].Poses), test.ShouldEqual, len(first))

	// Timestamps increase monotonically.
	test.That(t, a.Track[0].TMs, test.ShouldEqual, 0)
	test.That(t, a.Track[1].TMs, test.ShouldBeGreaterThan, a.Track[0].TMs)

	// The arm/tool actually moves: at least one frame's pose changes start→end.
	moved := false
	last := a.Track[len(a.Track)-1].Poses
	for name, p0 := range first {
		if pN, ok := last[name]; ok && (pN.X != p0.X || pN.Y != p0.Y || pN.Z != p0.Z) {
			moved = true
			break
		}
	}
	test.That(t, moved, test.ShouldBeTrue)

	// Scene includes both the arm and a known obstacle.
	var hasArm, hasObstacle bool
	for _, tf := range a.Scene.Transforms() {
		if strings.HasPrefix(tf.GetReferenceFrame(), "arm:") {
			hasArm = true
		}
		if tf.GetReferenceFrame() == "coffee-machine-base" {
			hasObstacle = true
		}
	}
	test.That(t, hasArm, test.ShouldBeTrue)
	test.That(t, hasObstacle, test.ShouldBeTrue)
}
