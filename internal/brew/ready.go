package brew

import "math"

func deg(d float64) float64 { return d * math.Pi / 180 }

// readyConfigs are the validated, collision-free "ready" joint configurations
// (radians) each arm starts from. A real arm always begins at a valid pose; the
// all-zero configuration folds these arms down through the table, leaving the
// planner no valid start. These standing configs were found by collision search
// and confirmed to plan the full brew sequence. Joint order:
// xarm6  [waist, shoulder, elbow, forearm_rot, wrist, gripper_rot]
// ur5e   [shoulder_pan, shoulder_lift, elbow, wrist_1, wrist_2, wrist_3]
var readyConfigs = map[string][]float64{
	"xarm6": {0, deg(-60), deg(-90), 0, 0, 0},
	"ur5e":  {0, deg(-90), deg(-90), 0, 0, 0},
}

// ReadyConfig returns the validated start joint configuration (radians) for the
// named arm, and whether one is defined.
func ReadyConfig(arm string) ([]float64, bool) {
	cfg, ok := readyConfigs[arm]
	return cfg, ok
}
