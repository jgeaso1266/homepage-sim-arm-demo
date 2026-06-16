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

// AllowedCollision names a pair of frames permitted to overlap during planning.
// It mirrors beanjamin's AllowedCollision and is used for legitimate
// tool-against-target contact (e.g. the portafilter handle reaching inside the
// grinder or the coffee-machine actuation area).
type AllowedCollision struct {
	Frame1 string
	Frame2 string
}

// Step is one named goal in the brew sequence. Pose is the world-frame target
// for the tool ("filter") frame. Linear requests a straight-line move.
// AllowedCollisions lists frame pairs that may overlap for this step only.
type Step struct {
	Name              string
	Pose              spatialmath.Pose
	Linear            bool
	AllowedCollisions []AllowedCollision
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
	// home: a comfortable pose above the workspace, tool pointing straight down,
	// on the +x side away from the camera mast (zoo-cam-obstacle at y=-740).
	home := ovDeg(350, 0, 300, 0, 0, -1, 0)

	grinderActivate := ovDeg(280, -540, 95, 0, -1, 0, -180)
	tamperActivate := ovDeg(615, -435.7, 112.3, 0.81, -0.59, 0, -180)
	coffeeIn := ovDeg(689.6, -12.45, 155, 0.66, -0.75, 0, -179)

	// At each station the tool (filter cup, its dangling portafilter handle, and
	// the claws) is *supposed* to be at/inside the station, so we allow the tool
	// frames to contact that station's geometry on both its approach and activate
	// steps — mirroring beanjamin's coffeeBrewingCollisions. The arm BODY links
	// are NOT exempted, so the arm still plans collision-free around the scene.
	grinder := toolVs("grinder-base", "grinder-mid", "grinder-top")
	tamper := toolVs("tamper-base", "tamper-mid", "tamper-top", "tamper-left", "tamper-right")
	coffee := toolVs(
		"coffee-machine-actuation-area", "coffee-machine-base", "coffee-machine-mid",
		"coffee-machine-top", "coffee-machine-buffer-front",
		"coffee-machine-buffer-left", "coffee-machine-buffer-right",
	)

	return []Step{
		{Name: "home", Pose: home},
		{Name: "grinder_approach", Pose: approach(grinderActivate), AllowedCollisions: grinder},
		{Name: "grinder_activate", Pose: grinderActivate, AllowedCollisions: grinder},
		{Name: "tamper_approach", Pose: approach(tamperActivate), AllowedCollisions: tamper},
		{Name: "tamper_activate", Pose: tamperActivate, AllowedCollisions: tamper},
		{Name: "coffee_approach", Pose: approach(coffeeIn), Linear: true, AllowedCollisions: coffee},
		{Name: "coffee_in", Pose: coffeeIn, Linear: true, AllowedCollisions: coffee},
	}
}

// toolParts are the gripper-mounted frames that legitimately contact a station.
var toolParts = []string{"filter", "portafilter-handle", "coffee-claws-middle"}

// toolVs returns allowed-collision pairs between every tool part and each named
// station frame, so the tool may contact the station it is acting on.
func toolVs(stationFrames ...string) []AllowedCollision {
	out := make([]AllowedCollision, 0, len(toolParts)*len(stationFrames))
	for _, t := range toolParts {
		for _, s := range stationFrames {
			out = append(out, AllowedCollision{Frame1: t, Frame2: s})
		}
	}
	return out
}
