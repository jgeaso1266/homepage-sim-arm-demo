package brew

import (
	"context"
	"fmt"

	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/motionplan"
	"go.viam.com/rdk/motionplan/armplanning"
	"go.viam.com/rdk/referenceframe"
)

// linearLineToleranceMm and linearOrientationToleranceDegs are the straight-line
// tolerances applied to steps marked Linear, mirroring beanjamin's
// defaultApproachConstraint.
const (
	linearLineToleranceMm          = 1.0
	linearOrientationToleranceDegs = 2.0
)

// PlanSequence plans the full brew sequence into a single concatenated joint
// trajectory. It mirrors beanjamin's moveToRawPose: each step plans a motion of
// the tool frame to the step's world-frame goal, starting from where the
// previous step ended, and the per-step trajectories are stitched together.
//
// Linear steps get a LinearConstraint so the tool drives in a straight line.
func PlanSequence(
	ctx context.Context,
	logger logging.Logger,
	fs *referenceframe.FrameSystem,
	armName, toolFrame string,
	startInputs referenceframe.FrameSystemInputs,
	steps []Step,
) ([]referenceframe.FrameSystemInputs, error) {
	prevInputs := startInputs

	var trajectory []referenceframe.FrameSystemInputs
	for _, step := range steps {
		var constraints *motionplan.Constraints
		if step.Linear || len(step.AllowedCollisions) > 0 {
			constraints = &motionplan.Constraints{}
			if step.Linear {
				constraints.LinearConstraint = []motionplan.LinearConstraint{
					{
						LineToleranceMm:          linearLineToleranceMm,
						OrientationToleranceDegs: linearOrientationToleranceDegs,
					},
				}
			}
			if len(step.AllowedCollisions) > 0 {
				allows := make([]motionplan.CollisionSpecificationAllowedFrameCollisions, len(step.AllowedCollisions))
				for i, ac := range step.AllowedCollisions {
					allows[i] = motionplan.CollisionSpecificationAllowedFrameCollisions{
						Frame1: ac.Frame1,
						Frame2: ac.Frame2,
					}
				}
				constraints.CollisionSpecification = []motionplan.CollisionSpecification{{Allows: allows}}
			}
		}

		goal := armplanning.NewPlanState(
			referenceframe.FrameSystemPoses{
				toolFrame: referenceframe.NewPoseInFrame(referenceframe.World, step.Pose),
			},
			nil,
		)

		plan, _, err := armplanning.PlanMotion(ctx, logger, &armplanning.PlanRequest{
			FrameSystem: fs,
			Goals:       []*armplanning.PlanState{goal},
			StartState:  armplanning.NewPlanState(nil, prevInputs),
			Constraints: constraints,
		})
		if err != nil {
			return nil, fmt.Errorf("plan step %q to %v: %w", step.Name, step.Pose.Point(), err)
		}

		stepTraj := plan.Trajectory()
		if len(stepTraj) == 0 {
			return nil, fmt.Errorf("plan step %q returned an empty trajectory", step.Name)
		}

		trajectory = append(trajectory, stepTraj...)
		prevInputs = stepTraj[len(stepTraj)-1]
	}

	return trajectory, nil
}
