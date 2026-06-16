package scene

import (
	"math"
	"sort"
	"testing"

	"go.viam.com/rdk/referenceframe"
	"go.viam.com/rdk/spatialmath"
	"go.viam.com/test"
)

func TestBuildFrameSystem_xarm6(t *testing.T) {
	fs, err := BuildFrameSystem("arm", "../../data/kinematics/xarm6.json", "../../data/beanjamin-config.merged.json")
	test.That(t, err, test.ShouldBeNil)

	names := fs.FrameNames()
	test.That(t, names, test.ShouldContain, "arm")                 // arm model
	test.That(t, names, test.ShouldContain, "filter")              // tool frame
	test.That(t, names, test.ShouldContain, "coffee-machine-base") // obstacle

	// Lock the reachability-critical tool chain: filter → gripper → arm.
	// Without this stack xArm6 cannot reach the tamper pose, so its structure
	// must not silently drift.
	filterParent, err := fs.Parent(fs.Frame("filter"))
	test.That(t, err, test.ShouldBeNil)
	test.That(t, filterParent.Name(), test.ShouldEqual, "gripper")

	gripperParent, err := fs.Parent(fs.Frame("gripper"))
	test.That(t, err, test.ShouldBeNil)
	test.That(t, gripperParent.Name(), test.ShouldEqual, "arm")

	// The tool stack extends the arm by a gripper (105mm off the flange) and a
	// filter tip (220mm beyond the gripper). At zero inputs the xArm6 flange
	// points downward, so each frame sits further along -Z than its parent; what
	// matters for reachability is the stacking *separation*, which must not
	// regress. Verify the world-Z separations at zero inputs.
	inputs := referenceframe.NewZeroInputs(fs).ToLinearInputs()
	zAt := func(frame string) float64 {
		t.Helper()
		tf, err := fs.Transform(
			inputs,
			referenceframe.NewPoseInFrame(frame, spatialmath.NewZeroPose()),
			referenceframe.World,
		)
		test.That(t, err, test.ShouldBeNil)
		return tf.(*referenceframe.PoseInFrame).Pose().Point().Z
	}

	armZ := zAt("arm")
	gripperZ := zAt("gripper")
	filterZ := zAt("filter")

	test.That(t, math.Abs(gripperZ-armZ), test.ShouldAlmostEqual, 105.0, 1.0)
	test.That(t, math.Abs(filterZ-gripperZ), test.ShouldAlmostEqual, 220.0, 1.0)

	// Lock obstacle parenting: the coffee-machine mid box hangs off its base.
	midParent, err := fs.Parent(fs.Frame("coffee-machine-mid"))
	test.That(t, err, test.ShouldBeNil)
	test.That(t, midParent.Name(), test.ShouldEqual, "coffee-machine-base")

	// Print the full sorted scene so we can eyeball it.
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	t.Logf("xarm6 frame system (%d frames): %v", len(sorted), sorted)
}

func TestBuildFrameSystem_ur5e(t *testing.T) {
	fs, err := BuildFrameSystem("arm", "../../data/kinematics/ur5e.json", "../../data/beanjamin-config.merged.json")
	test.That(t, err, test.ShouldBeNil)

	names := fs.FrameNames()
	test.That(t, names, test.ShouldContain, "arm")
	test.That(t, names, test.ShouldContain, "filter")
	test.That(t, names, test.ShouldContain, "coffee-machine-base")
}
