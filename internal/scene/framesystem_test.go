package scene

import (
	"sort"
	"testing"

	"go.viam.com/test"
)

func TestBuildFrameSystem_xarm6(t *testing.T) {
	fs, err := BuildFrameSystem("arm", "../../data/kinematics/xarm6.json", "../../data/beanjamin-config.merged.json")
	test.That(t, err, test.ShouldBeNil)

	names := fs.FrameNames()
	test.That(t, names, test.ShouldContain, "arm")                 // arm model
	test.That(t, names, test.ShouldContain, "filter")              // tool frame
	test.That(t, names, test.ShouldContain, "coffee-machine-base") // obstacle

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
