package scene

import (
	"fmt"

	"github.com/golang/geo/r3"
	"go.viam.com/rdk/referenceframe"
	"go.viam.com/rdk/spatialmath"
)

// BuildFrameSystem assembles the planning frame system the baker plans against:
// the arm model, the Beanjamin tool chain (gripper → filter → portafilter-handle
// plus the coffee-claws-middle gripper geometry), and every world-scene obstacle
// from the machine config.
//
// The tool chain is the spike's key finding: without modelling the ~325mm
// gripper+filter stack, xArm6 cannot reach the tamper pose. Planning goals are
// keyed to the "filter" frame at the tool tip.
func BuildFrameSystem(armName, modelPath, configPath string) (*referenceframe.FrameSystem, error) {
	model, err := referenceframe.ParseModelJSONFile(modelPath, armName)
	if err != nil {
		return nil, fmt.Errorf("parse model %s: %w", modelPath, err)
	}

	fs := referenceframe.NewEmptyFrameSystem("fs")
	if err := fs.AddFrame(model, fs.World()); err != nil {
		return nil, fmt.Errorf("add arm frame: %w", err)
	}

	if err := addToolChain(fs, model, configPath); err != nil {
		return nil, err
	}
	if err := addObstacles(fs, configPath); err != nil {
		return nil, err
	}
	return fs, nil
}

// addToolChain mounts the gripper/filter/portafilter tool stack on the arm model.
func addToolChain(fs *referenceframe.FrameSystem, model referenceframe.Frame, configPath string) error {
	// The frames built below must exactly match the shared tool-frame set that
	// LoadObstacles uses to exclude tool-chain frames from the world scene. Guard
	// against the two drifting: if a frame is added/removed here it must also be
	// reflected in toolFrameNames, and vice-versa.
	built := []string{"gripper", "coffee-claws-middle", "filter", "portafilter-handle"}
	if len(built) != len(toolFrameNames) {
		return fmt.Errorf("tool-chain frames %v out of sync with toolFrameNames %v", built, toolFrameNames)
	}
	for _, name := range built {
		if !toolFrameNames[name] {
			return fmt.Errorf("tool-chain frame %q missing from toolFrameNames", name)
		}
	}

	// gripper: z=105 above the flange. This translation is not in the machine
	// config; it comes from the beanjamin fragment mod.
	gripper, err := referenceframe.NewStaticFrame("gripper", spatialmath.NewPoseFromPoint(r3.Vector{X: 0, Y: 0, Z: 105})) // from beanjamin fragment mod
	if err != nil {
		return fmt.Errorf("create gripper frame: %w", err)
	}
	if err := fs.AddFrame(gripper, model); err != nil {
		return fmt.Errorf("add gripper frame: %w", err)
	}

	// The ufactory gripper's own body geometry — its case and claws — comes from
	// the gripper module's kinematics, NOT the machine config, so it isn't in the
	// config export. These two boxes (parent "gripper", local offsets back toward
	// the flange) fill the otherwise-empty space between the arm flange and the
	// held portafilter. Dimensions/offsets read from the live Beanjamin viz.
	if err := addGripperGeometry(fs, gripper, "case-gripper", -50, r3.Vector{X: 50, Y: 100, Z: 100}); err != nil {
		return err
	}
	if err := addGripperGeometry(fs, gripper, "claws", -2.5, r3.Vector{X: 40, Y: 170, Z: 105}); err != nil {
		return err
	}

	// Each tool-chain frame's transform + geometry comes straight from the config.
	// coffee-claws-middle: gripper-attached collision box (parent gripper).
	claws, err := LoadToolFrame(configPath, "coffee-claws-middle")
	if err != nil {
		return err
	}
	if _, err := addPart(fs, claws, gripper); err != nil {
		return err
	}

	// filter: sphere tool tip (parent gripper) — the planning target frame.
	filter, err := LoadToolFrame(configPath, "filter")
	if err != nil {
		return err
	}
	filterFrame, err := addPart(fs, filter, gripper)
	if err != nil {
		return err
	}

	// portafilter-handle: box hanging below the filter (parent the filter frame).
	handle, err := LoadToolFrame(configPath, "portafilter-handle")
	if err != nil {
		return err
	}
	if _, err := addPart(fs, handle, filterFrame); err != nil {
		return err
	}
	return nil
}

// addGripperGeometry attaches one of the gripper module's body boxes to the
// gripper frame at local (0, 0, zMM), as a part (so it renders/collides at the
// right place via addPart's two-frame handling).
func addGripperGeometry(fs *referenceframe.FrameSystem, gripper referenceframe.Frame, name string, zMM float64, dims r3.Vector) error {
	box, err := spatialmath.NewBox(spatialmath.NewZeroPose(), dims, name)
	if err != nil {
		return fmt.Errorf("create %s geometry: %w", name, err)
	}
	part := Obstacle{
		Name:        name,
		Parent:      "gripper",
		Translation: r3.Vector{X: 0, Y: 0, Z: zMM},
		Orientation: &spatialmath.OrientationVectorDegrees{OZ: 1},
		Geometry:    box,
	}
	if _, err := addPart(fs, part, gripper); err != nil {
		return err
	}
	return nil
}

// geomFrameSuffix names the child frame that carries a part's geometry.
const geomFrameSuffix = ":geometry"

// addPart adds a config part (obstacle or tool frame) as TWO frames under parent:
//
//   - a transform frame named o.Name carrying the part's pose — this is the
//     kinematic node children attach to (and, for the filter, the planning target);
//   - a zero-offset child frame "o.Name:geometry" carrying the geometry.
//
// rdk positions a static frame's own geometry at the frame's PARENT, not at the
// frame itself (the link convention: a frame's transform moves its children, not
// its own geometry). Attaching the geometry to a zero-offset child therefore
// renders/collides it at o.Name's true world position. The geometry keeps its
// o.Name label, so the emitted transform's referenceFrame is unchanged. Returns
// the transform frame so callers can parent children to it.
func addPart(fs *referenceframe.FrameSystem, o Obstacle, parent referenceframe.Frame) (referenceframe.Frame, error) {
	tf, err := referenceframe.NewStaticFrame(o.Name, spatialmath.NewPose(o.Translation, o.Orientation))
	if err != nil {
		return nil, fmt.Errorf("create %s frame: %w", o.Name, err)
	}
	if err := fs.AddFrame(tf, parent); err != nil {
		return nil, fmt.Errorf("add %s frame: %w", o.Name, err)
	}
	gf, err := referenceframe.NewStaticFrameWithGeometry(o.Name+geomFrameSuffix, spatialmath.NewZeroPose(), o.Geometry)
	if err != nil {
		return nil, fmt.Errorf("create %s geometry frame: %w", o.Name, err)
	}
	if err := fs.AddFrame(gf, tf); err != nil {
		return nil, fmt.Errorf("add %s geometry frame: %w", o.Name, err)
	}
	return tf, nil
}

// peripheralCollisionExclusions are real config obstacles that sit outside the
// brew workspace (a camera mast, speaker, stream-deck, a stray cup). They are
// excluded from the PLANNING collision model: our generic rdk arm kinematics
// sweep into them, whereas Beanjamin's actual arm+gripper clears them. The arm
// body still plans collision-free against every interaction station (coffee
// machine, grinders, tamper) and structure (table, ceiling, mount). They are
// also omitted from the rendered scene (which is built from this same frame
// system) so the arm — planned without them — never appears to clip them.
var peripheralCollisionExclusions = map[string]bool{
	"zoo-cam-obstacle":     true,
	"speaker-obstacle":     true,
	"stream-deck-obstacle": true,
	"empty-cup":            true,
	// Bounds that are useful on the real machine but visually intrusive and
	// unnecessary for this short tabletop brew (the stations are all low): the
	// ceiling is a slab at z=900 and the wall a 2x2m plane behind the arm.
	"ceiling": true,
	"wall":    true,
}

// addObstacles attaches every world-scene obstacle from the config in an order
// that guarantees each frame's parent is already present.
func addObstacles(fs *referenceframe.FrameSystem, configPath string) error {
	loaded, err := LoadObstacles(configPath)
	if err != nil {
		return fmt.Errorf("load obstacles: %w", err)
	}

	obstacles := make(map[string]Obstacle, len(loaded))
	for name, o := range loaded {
		if peripheralCollisionExclusions[name] {
			continue // peripheral, non-interaction obstacle — not in collision model
		}
		obstacles[name] = o
	}

	added := make(map[string]bool, len(obstacles))
	remaining := len(obstacles)
	for remaining > 0 {
		progressed := false
		for name, o := range obstacles {
			if added[name] {
				continue
			}
			parent := fs.Frame(o.Parent)
			if parent == nil {
				continue // parent not yet present; try again next pass
			}
			if _, err := addPart(fs, o, parent); err != nil {
				return fmt.Errorf("add obstacle %s: %w", name, err)
			}
			added[name] = true
			remaining--
			progressed = true
		}
		if !progressed {
			// Every remaining obstacle references a parent that never appears.
			missing := make([]string, 0, remaining)
			for name := range obstacles {
				if !added[name] {
					missing = append(missing, fmt.Sprintf("%s(parent=%s)", name, obstacles[name].Parent))
				}
			}
			return fmt.Errorf("obstacles reference unknown parents: %v", missing)
		}
	}
	return nil
}

// frameFor builds a static frame carrying the obstacle's geometry, posed at its
// translation + orientation relative to its parent.
