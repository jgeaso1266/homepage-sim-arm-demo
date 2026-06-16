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

// tickMs is the spacing between trajectory steps during playback.
const tickMs = 40

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

	// Scene snapshot: every geometry (obstacles + arm/tool) at the start pose.
	// Pin the snapshot UUID so transform identities are stable across re-bakes
	// (DrawFrameSystemGeometries derives every transform UUID from it); otherwise a
	// fresh random UUID would churn every entity id on each run. (Track poses are
	// rounded but may still differ sub-micron between bakes — that's expected.)
	// Bake a camera framed on the workspace (arm at origin; grinder/tamper/machine
	// out to +x/-y), so the embedded view is well-composed without runtime tuning.
	colors := sceneColors(fs)
	camera := draw.NewSceneCamera(
		r3.Vector{X: 2200, Y: -1950, Z: 1500},
		r3.Vector{X: 360, Y: -180, Z: 520},
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
		track[i] = TrackStep{TMs: i * tickMs, Poses: poses}
	}

	return &Asset{Scene: snap, Track: track}, nil
}

// Moving geometries are the arm's own links (named "arm:arm:<link>") and the
// gripper-mounted tool frames (named by their bare frame name). isMoving matches
// the transform ReferenceFrame against both forms.
var movingToolFrames = map[string]bool{
	"filter":              true,
	"portafilter-handle":  true,
	"coffee-claws-middle": true,
}

func isMoving(referenceFrame string) bool {
	return strings.HasPrefix(referenceFrame, armFrameName+":") || movingToolFrames[referenceFrame]
}

// round quantizes a pose component to 0.01mm / 0.01°, well below visual
// resolution, so sub-micron planner jitter doesn't churn the committed assets.
func round(v float64) float64 { return math.Round(v*100) / 100 }

// sceneColors tints the arm/tool geometries distinctly from the static scene.
func sceneColors(fs *referenceframe.FrameSystem) map[string]draw.Color {
	arm := draw.ColorFromHex("#2DD4BF")   // teal — the robot
	scene := draw.ColorFromHex("#94A3B8") // slate — the workspace
	colors := make(map[string]draw.Color)
	for _, name := range fs.FrameNames() {
		// Source the tool-frame set from movingToolFrames so the color map can't
		// drift from isMoving. "gripper" is a pure (geometry-less) frame, included
		// here for completeness.
		if name == armFrameName || name == "gripper" || movingToolFrames[name] {
			colors[name] = arm
		} else {
			colors[name] = scene
		}
	}
	return colors
}

type unknownArmError struct{ arm string }

func (e *unknownArmError) Error() string { return "no ready config for arm " + e.arm }
