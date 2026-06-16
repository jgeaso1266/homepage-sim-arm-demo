// Package brew defines Beanjamin's espresso brew sequence as an ordered list of
// end-effector goal poses, ready to be planned into a joint trajectory.
package brew

import (
	"github.com/golang/geo/r3"
	"go.viam.com/rdk/spatialmath"
)

// standoffMM is how far the arm backs off along the tool's approach axis (local
// -Z of the goal orientation) before driving straight in to each activate pose.
const standoffMM = 80.0

// Step is one named goal in the brew sequence. Pose is the world-frame target
// for the tool ("filter") frame. Linear requests a straight-line move.
type Step struct {
	Name   string
	Pose   spatialmath.Pose
	Linear bool
}

// ovDeg builds a pose from a point (mm) and an orientation vector in degrees.
func ovDeg(x, y, z, ox, oy, oz, th float64) spatialmath.Pose {
	return spatialmath.NewPose(
		r3.Vector{X: x, Y: y, Z: z},
		&spatialmath.OrientationVectorDegrees{OX: ox, OY: oy, OZ: oz, Theta: th},
	)
}

// approach backs a goal pose off by standoffMM along the goal's local -Z, so the
// arm reaches the goal in a straight line along the tool's approach axis.
func approach(goal spatialmath.Pose) spatialmath.Pose {
	return spatialmath.Compose(goal, spatialmath.NewPoseFromPoint(r3.Vector{Z: -standoffMM}))
}

// Sequence returns the brew sequence in execution order. Absolute activate poses
// are taken from the Beanjamin machine config (world frame, mm / ov-degrees);
// each *_approach is derived as a standoff from its goal, never hardcoded.
func Sequence() []Step {
	// home: a comfortable pose above the workspace, tool pointing straight down.
	home := ovDeg(300, 0, 500, 0, 0, -1, 0)

	grinderActivate := ovDeg(280, -540, 95, 0, -1, 0, -180)
	tamperActivate := ovDeg(615, -435.7, 112.3, 0.81, -0.59, 0, -180)
	coffeeIn := ovDeg(689.6, -12.45, 155, 0.66, -0.75, 0, -179)

	return []Step{
		{Name: "home", Pose: home},
		{Name: "grinder_approach", Pose: approach(grinderActivate)},
		{Name: "grinder_activate", Pose: grinderActivate},
		{Name: "tamper_approach", Pose: approach(tamperActivate)},
		{Name: "tamper_activate", Pose: tamperActivate},
		{Name: "coffee_approach", Pose: approach(coffeeIn), Linear: true},
		{Name: "coffee_in", Pose: coffeeIn, Linear: true},
	}
}
