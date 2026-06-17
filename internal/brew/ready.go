package brew

import "math"

func deg(d float64) float64 { return d * math.Pi / 180 }

// readyConfigs are the "ready" joint configurations (radians) each arm starts
// from: a natural barista standby with the tool pointing straight down over the
// workspace (the home pose at ~(350,0,300)), rather than a stiff arm-up pose.
// Obtained by solving IK for that home pose per arm; collision-free and confirmed
// to plan the full brew sequence. Joint order:
// xarm6  [waist, shoulder, elbow, forearm_rot, wrist, gripper_rot]
// ur5e   [shoulder_pan, shoulder_lift, elbow, wrist_1, wrist_2, wrist_3]
var readyConfigs = map[string][]float64{
	"xarm6": {0, deg(-17.5), deg(-88.2), 0, deg(105.8), 0},
	"ur5e":  {deg(22.4), deg(-97.8), deg(-61.2), deg(68.9), deg(-90), deg(292.4)},
}

// ReadyConfig returns the validated start joint configuration (radians) for the
// named arm, and whether one is defined.
func ReadyConfig(arm string) ([]float64, bool) {
	cfg, ok := readyConfigs[arm]
	return cfg, ok
}
