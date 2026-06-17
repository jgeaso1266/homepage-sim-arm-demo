package brew

import "math"

func deg(d float64) float64 { return d * math.Pi / 180 }

// readyConfigs are the "ready" joint configurations (radians) each arm starts
// from. They place the gripper at Beanjamin's actual home pose — world
// (190.3, 399.74, 120.05) mm, orientation vector (1, 0, 0) at theta -180° (the
// gripper held horizontal, pointing +X, low over the bench) — obtained by solving
// IK for that gripper pose per arm. Collision-free and confirmed to plan the full
// brew sequence. Joint order:
// xarm6  [waist, shoulder, elbow, forearm_rot, wrist, gripper_rot]
// ur5e   [shoulder_pan, shoulder_lift, elbow, wrist_1, wrist_2, wrist_3]
var readyConfigs = map[string][]float64{
	"xarm6": {deg(91.8), deg(29.5), deg(-49.1), deg(-88.3), deg(90.6), deg(-19.6)},
	"ur5e":  {deg(-107.4), deg(-66.5), deg(124.5), deg(-58.0), deg(162.6), deg(-270.0)},
}

// ReadyConfig returns the validated start joint configuration (radians) for the
// named arm, and whether one is defined.
func ReadyConfig(arm string) ([]float64, bool) {
	cfg, ok := readyConfigs[arm]
	return cfg, ok
}
