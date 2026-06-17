package brew

import (
	"context"
	"testing"

	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/referenceframe"
	"go.viam.com/test"

	"homepage-simulated-arm-demo/internal/scene"
)

// TestPlanSequence_bothArms is the real reachability gate: it builds the
// obstacle-aware planning frame system for each candidate arm and plans the full
// brew sequence end-to-end, asserting every step planned a non-empty trajectory.
func TestPlanSequence_bothArms(t *testing.T) {
	for _, arm := range []string{"xarm6", "ur5e"} {
		fs, err := scene.BuildFrameSystem(
			"arm",
			"../../data/kinematics/"+arm+".json",
			"../../data/beanjamin-config.merged.json",
		)
		test.That(t, err, test.ShouldBeNil)

		cfg, ok := ReadyConfig(arm)
		test.That(t, ok, test.ShouldBeTrue)
		start := referenceframe.NewZeroInputs(fs)
		start["arm"] = cfg

		planned, err := PlanSequence(context.Background(), logging.NewTestLogger(t), fs, "arm", "filter", start, Sequence())
		test.That(t, err, test.ShouldBeNil)
		test.That(t, len(planned), test.ShouldEqual, len(Sequence()))
		for _, ps := range planned {
			test.That(t, len(ps.Traj), test.ShouldBeGreaterThan, 0)
		}
	}
}
