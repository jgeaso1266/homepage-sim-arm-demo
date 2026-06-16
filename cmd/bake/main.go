// Command bake is, for now, a SPIKE: it verifies that rdk's armplanning can
// plan Beanjamin's brew-sequence goal poses for both candidate arms (xArm6 and
// UR5e) from a standalone module. It retires three risks before we commit to the
// full baker:
//   - the rdk armplanning API resolves and runs outside rdk,
//   - both arms can IK-reach the brew poses (reachability), and
//   - PlanMotion returns a non-empty trajectory.
//
// Obstacles and sequence-chaining are intentionally omitted here; this is a
// reachability probe in an empty world.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/geo/r3"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/motionplan/armplanning"
	"go.viam.com/rdk/referenceframe"
	"go.viam.com/rdk/spatialmath"
)

// goal is one named end-effector target from Beanjamin's brew sequence,
// taken from the machine config (world frame, millimeters / ov-degrees).
type goal struct {
	name string
	pose spatialmath.Pose
}

func ovDeg(x, y, z, ox, oy, oz, th float64) spatialmath.Pose {
	return spatialmath.NewPose(
		r3.Vector{X: x, Y: y, Z: z},
		&spatialmath.OrientationVectorDegrees{OX: ox, OY: oy, OZ: oz, Theta: th},
	)
}

// brewGoals are the reach-stressing poses from the Beanjamin config.
var brewGoals = []goal{
	{"grinder_activate", ovDeg(280, -540, 95, 0, -1, 0, -180)},
	{"tamper_activate", ovDeg(615, -435.7, 112.3, 0.81, -0.59, 0, -180)},
	{"coffee_in", ovDeg(689.6, -12.45, 155, 0.66, -0.75, 0, -179)},
}

func planArm(ctx context.Context, logger logging.Logger, armName, modelPath string) error {
	model, err := referenceframe.ParseModelJSONFile(modelPath, armName)
	if err != nil {
		return fmt.Errorf("parse model %s: %w", modelPath, err)
	}

	fs := referenceframe.NewEmptyFrameSystem(armName + "-fs")
	if err := fs.AddFrame(model, fs.World()); err != nil {
		return fmt.Errorf("add arm frame: %w", err)
	}

	// Tool chain from the Beanjamin config: gripper (z=105 above flange) + filter
	// (z=220 above gripper) ⇒ the planning target is the filter tip, ~325mm beyond
	// the flange. Goals are keyed to this frame, matching real Beanjamin.
	const toolFrame = "filter"
	tool, err := referenceframe.NewStaticFrame(toolFrame, spatialmath.NewPoseFromPoint(r3.Vector{X: 0, Y: 0, Z: 325}))
	if err != nil {
		return fmt.Errorf("create tool frame: %w", err)
	}
	if err := fs.AddFrame(tool, model); err != nil {
		return fmt.Errorf("add tool frame: %w", err)
	}

	start := armplanning.NewPlanState(nil, referenceframe.NewZeroInputs(fs))

	fmt.Printf("\n=== %s (%s) — %d DOF, target=%s ===\n", armName, filepath.Base(modelPath), len(model.DoF()), toolFrame)
	for _, g := range brewGoals {
		goalState := armplanning.NewPlanState(
			referenceframe.FrameSystemPoses{toolFrame: referenceframe.NewPoseInFrame(referenceframe.World, g.pose)},
			nil,
		)
		plan, _, err := armplanning.PlanMotion(ctx, logger, &armplanning.PlanRequest{
			FrameSystem: fs,
			Goals:       []*armplanning.PlanState{goalState},
			StartState:  start,
		})
		if err != nil {
			fmt.Printf("  ✗ %-18s UNREACHABLE: %v\n", g.name, err)
			continue
		}
		fmt.Printf("  ✓ %-18s planned: %d trajectory steps\n", g.name, len(plan.Trajectory()))
	}
	return nil
}

func main() {
	ctx := context.Background()
	logger := logging.NewLogger("bake-spike")

	kdir := "data/kinematics"
	arms := []struct{ name, path string }{
		{"arm", filepath.Join(kdir, "xarm6.json")},
		{"arm", filepath.Join(kdir, "ur5e.json")},
	}
	for _, a := range arms {
		if err := planArm(ctx, logger, a.name, a.path); err != nil {
			fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
			os.Exit(1)
		}
	}
}
