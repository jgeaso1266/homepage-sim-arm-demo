package brew

import (
	"testing"

	"go.viam.com/rdk/spatialmath"
	"go.viam.com/test"
)

func TestBrewSequence(t *testing.T) {
	seq := Sequence()
	names := make([]string, len(seq))
	for i, s := range seq {
		names[i] = s.Name
	}
	test.That(t, names, test.ShouldResemble, []string{
		"home",
		"grinder_approach", "grinder_activate", "grinder_retract",
		"tamper_approach", "tamper_activate", "tamper_retract",
		"coffee_approach", "coffee_in",
	})

	// coffee_in absolute pose from config (final step).
	ci := seq[len(seq)-1]
	test.That(t, ci.Name, test.ShouldEqual, "coffee_in")
	test.That(t, ci.Pose.Point().X, test.ShouldAlmostEqual, 689.6)

	// grinder_approach is derived as a standoff from grinder_activate: its point
	// differs and the straight-line distance between them is ~80mm.
	var approach, activate Step
	for _, s := range seq {
		switch s.Name {
		case "grinder_approach":
			approach = s
		case "grinder_activate":
			activate = s
		}
	}
	test.That(t, spatialmath.PoseAlmostCoincident(approach.Pose, activate.Pose), test.ShouldBeFalse)
	standoff := activate.Pose.Point().Sub(approach.Pose.Point()).Norm()
	test.That(t, standoff, test.ShouldAlmostEqual, 80.0)

	// The straight drive-in (coffee_in) is linear; its approach is a free move.
	for _, s := range seq {
		switch s.Name {
		case "coffee_in":
			test.That(t, s.Linear, test.ShouldBeTrue)
		case "coffee_approach":
			test.That(t, s.Linear, test.ShouldBeFalse)
		}
	}
}
