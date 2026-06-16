package scene

import (
	"testing"

	"go.viam.com/test"
)

func TestLoadObstacles(t *testing.T) {
	obs, err := LoadObstacles("../../data/beanjamin-config.merged.json")
	test.That(t, err, test.ShouldBeNil)

	// coffee-machine-base is a world-parented box at x=740 (from config).
	cm, ok := obs["coffee-machine-base"]
	test.That(t, ok, test.ShouldBeTrue)
	test.That(t, cm.Parent, test.ShouldEqual, "world")
	test.That(t, cm.Translation.X, test.ShouldAlmostEqual, 740.0)

	// nested obstacle keeps its parent (e.g. coffee-machine-mid -> coffee-machine-base).
	mid, ok := obs["coffee-machine-mid"]
	test.That(t, ok, test.ShouldBeTrue)
	test.That(t, mid.Parent, test.ShouldEqual, "coffee-machine-base")

	// gripper-attached frames (tool chain) are handled in a later task, not here.
	_, ok = obs["filter"]
	test.That(t, ok, test.ShouldBeFalse)
}
