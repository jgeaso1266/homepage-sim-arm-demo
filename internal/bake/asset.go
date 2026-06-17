// Package bake turns a planned brew trajectory into the static assets the web
// app replays: a scene Snapshot (obstacles + arm at its start pose) and a track
// of per-step world-frame poses for the moving arm/tool geometries.
package bake

import (
	"context"
	"math"
	"strings"

	"github.com/golang/geo/r3"
	"github.com/google/uuid"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/referenceframe"

	"github.com/viam-labs/motion-tools/draw"

	"homepage-simulated-arm-demo/internal/brew"
	"homepage-simulated-arm-demo/internal/scene"
)

// sceneUUID is a fixed snapshot identity so re-baking produces byte-stable
// assets (transform UUIDs are derived from it).
var sceneUUID = uuid.MustParse("be4a0000-0000-4000-8000-000000000001")

// armFrameName is the frame name the arm model is mounted under in the planning
// frame system; its links and the gripper-mounted tool frames are what move.
const armFrameName = "arm"

// brewDurationMs is the wall-clock length of a full brew playback, and brewFrames
// is how many evenly-spaced frames it's resampled to. The planner samples
// unevenly (dense at goals, sparse in transit); resampling to even joint-space
// spacing + uniform per-frame timing yields constant-speed, smooth playback.
const (
	brewDurationMs = 6000.0
	brewFrames     = 180
)

// Pose is a world-frame pose in motion-tools' common.v1.Pose JSON shape
// (millimeters + orientation vector degrees), so the web side can feed it
// straight into poseToMatrix.
type Pose struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Z     float64 `json:"z"`
	OX    float64 `json:"o_x"`
	OY    float64 `json:"o_y"`
	OZ    float64 `json:"o_z"`
	Theta float64 `json:"theta"`
}

// TrackStep is one playback frame: the world pose of every moving entity, keyed
// by the entity's "<frame>:<geometryLabel>" name (matching the scene snapshot).
type TrackStep struct {
	TMs   int             `json:"tMs"`
	Poses map[string]Pose `json:"poses"`
}

// Asset is the full replay payload for one arm.
type Asset struct {
	Scene *draw.Snapshot
	Track []TrackStep
}

// Baker builds assets from data files. Paths are injected so the same code runs
// from the repo root (cmd/bake) and from a test working directory.
type Baker struct {
	KinematicsDir string // directory holding <arm>.json kinematics models
	ConfigPath    string // path to the merged beanjamin config
}

// Build plans the brew sequence for arm and returns its replay Asset.
func (b Baker) Build(ctx context.Context, logger logging.Logger, arm string) (*Asset, error) {
	fs, err := scene.BuildFrameSystem(armFrameName, b.KinematicsDir+"/"+arm+".json", b.ConfigPath)
	if err != nil {
		return nil, err
	}

	cfg, ok := brew.ReadyConfig(arm)
	if !ok {
		return nil, &unknownArmError{arm}
	}
	start := referenceframe.NewZeroInputs(fs)
	start[armFrameName] = cfg

	traj, err := brew.PlanSequence(ctx, logger, fs, armFrameName, "filter", start, brew.Sequence())
	if err != nil {
		return nil, err
	}
	// Resample to frames evenly spaced along the joint-space arc, so uniform
	// per-frame timing plays at a constant speed and tweens smoothly across the
	// planner's sparse transit gaps (which would otherwise freeze-then-jump).
	traj = resampleByDistance(traj, armFrameName, brewFrames)

	// Scene snapshot: every geometry (obstacles + arm/tool) at the start pose.
	// Pin the snapshot UUID so transform identities are stable across re-bakes
	// (DrawFrameSystemGeometries derives every transform UUID from it); otherwise a
	// fresh random UUID would churn every entity id on each run. (Track poses are
	// rounded but may still differ sub-micron between bakes — that's expected.)
	// Bake a camera framed on the workspace (arm at origin; grinder/tamper/machine
	// out to +x/-y), so the embedded view is well-composed without runtime tuning.
	colors := sceneColors(fs)
	camera := draw.NewSceneCamera(
		r3.Vector{X: 1850, Y: -1600, Z: 1200},
		r3.Vector{X: 340, Y: -120, Z: 180},
	)
	snap := draw.NewSnapshot(draw.WithSceneCamera(camera))
	snap.SetUUID(sceneUUID)
	if err := snap.DrawFrameSystemGeometries(fs, traj[0], colors); err != nil {
		return nil, err
	}

	// Track: per step, the world pose of each moving (arm/tool) geometry.
	track := make([]TrackStep, len(traj))
	for i, inputs := range traj {
		transforms, err := draw.NewDrawnFrameSystem(fs, inputs).ToTransforms()
		if err != nil {
			return nil, err
		}
		poses := make(map[string]Pose)
		for _, t := range transforms {
			if !isMoving(t.GetReferenceFrame()) {
				continue
			}
			// The scene's observer frame is World, so each geometry's world pose
			// lives in its center (PhysicalObject), not PoseInObserverFrame.
			p := t.GetPhysicalObject().GetCenter()
			poses[t.GetReferenceFrame()] = Pose{
				X: round(p.GetX()), Y: round(p.GetY()), Z: round(p.GetZ()),
				OX: round(p.GetOX()), OY: round(p.GetOY()), OZ: round(p.GetOZ()), Theta: round(p.GetTheta()),
			}
		}
		tMs := 0
		if len(traj) > 1 {
			tMs = int(math.Round(float64(i) / float64(len(traj)-1) * brewDurationMs))
		}
		track[i] = TrackStep{TMs: tMs, Poses: poses}
	}

	return &Asset{Scene: snap, Track: track}, nil
}

// jointDistance is the L2 distance between two arm configurations. Inputs are
// joint values (radians); referenceframe.Input is an alias for float64.
func jointDistance(a, b []referenceframe.Input) float64 {
	sum := 0.0
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return math.Sqrt(sum)
}

// resampleByDistance returns numFrames arm configurations spaced evenly along the
// trajectory's joint-space arc length (linear interpolation within each planned
// segment). This decouples playback speed from the planner's uneven sampling:
// with even spacing, uniform per-frame timing renders constant-speed motion.
func resampleByDistance(traj []referenceframe.FrameSystemInputs, armName string, numFrames int) []referenceframe.FrameSystemInputs {
	if len(traj) <= 1 || numFrames <= 1 {
		return traj
	}
	cum := make([]float64, len(traj))
	for i := 1; i < len(traj); i++ {
		cum[i] = cum[i-1] + jointDistance(traj[i-1][armName], traj[i][armName])
	}
	total := cum[len(cum)-1]
	if total == 0 {
		return traj
	}

	out := make([]referenceframe.FrameSystemInputs, numFrames)
	seg := 0
	for k := 0; k < numFrames; k++ {
		target := float64(k) / float64(numFrames-1) * total
		for seg < len(cum)-2 && cum[seg+1] < target {
			seg++
		}
		segLen := cum[seg+1] - cum[seg]
		f := 0.0
		if segLen > 0 {
			f = (target - cum[seg]) / segLen
		}
		out[k] = lerpInputs(traj[seg], traj[seg+1], f, armName)
	}
	return out
}

// lerpInputs linearly interpolates the arm configuration between a and b by f,
// carrying over the (input-less) static frames unchanged.
func lerpInputs(a, b referenceframe.FrameSystemInputs, f float64, armName string) referenceframe.FrameSystemInputs {
	out := referenceframe.FrameSystemInputs{}
	for k, v := range a {
		out[k] = v
	}
	ja, jb := a[armName], b[armName]
	arm := make([]referenceframe.Input, len(ja))
	for i := range ja {
		arm[i] = ja[i] + (jb[i]-ja[i])*f
	}
	out[armName] = arm
	return out
}

// toolBases are the gripper-mounted parts that move with the arm. Their geometry
// is emitted on a "<base>:geometry" child frame (see scene.addPart), so the
// rendered transform's referenceFrame is "<base>:geometry:<base>". "gripper" is a
// pure (geometry-less) mount frame, listed for color completeness.
var toolBases = []string{
	"gripper", "filter", "portafilter-handle", "coffee-claws-middle", "case-gripper", "claws",
}

// isArmOrTool reports whether a name belongs to the moving robot — the arm model
// chain or a gripper-mounted part. It works for both bare frame names ("filter",
// "filter:geometry") and emitted transform referenceFrames ("filter:geometry:filter",
// "arm:arm:upper_arm"), since every such name is the base or "<base>:…".
func isArmOrTool(name string) bool {
	if name == armFrameName || strings.HasPrefix(name, armFrameName+":") {
		return true
	}
	for _, b := range toolBases {
		if name == b || strings.HasPrefix(name, b+":") {
			return true
		}
	}
	return false
}

func isMoving(referenceFrame string) bool { return isArmOrTool(referenceFrame) }

// round quantizes a pose component to 0.01mm / 0.01°, well below visual
// resolution, so sub-micron planner jitter doesn't churn the committed assets.
func round(v float64) float64 { return math.Round(v*100) / 100 }

// sceneColors tints the arm/tool geometries distinctly from the static scene.
func sceneColors(fs *referenceframe.FrameSystem) map[string]draw.Color {
	arm := draw.ColorFromHex("#2DD4BF")   // teal — the robot
	scene := draw.ColorFromHex("#94A3B8") // slate — the workspace
	colors := make(map[string]draw.Color)
	for _, name := range fs.FrameNames() {
		// Same predicate as the track's isMoving, so color and motion never drift.
		if isArmOrTool(name) {
			colors[name] = arm
		} else {
			colors[name] = scene
		}
	}
	return colors
}

type unknownArmError struct{ arm string }

func (e *unknownArmError) Error() string { return "no ready config for arm " + e.arm }
