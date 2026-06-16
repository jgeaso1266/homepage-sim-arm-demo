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
	// config; it comes from the beanjamin fragment mod. The gripper carries no
	// own geometry, so the coffee-claws-middle box is added as a separate child.
	gripper, err := referenceframe.NewStaticFrame("gripper", spatialmath.NewPoseFromPoint(r3.Vector{X: 0, Y: 0, Z: 105})) // from beanjamin fragment mod
	if err != nil {
		return fmt.Errorf("create gripper frame: %w", err)
	}
	if err := fs.AddFrame(gripper, model); err != nil {
		return fmt.Errorf("add gripper frame: %w", err)
	}

	// Each tool-chain frame's transform + geometry comes straight from the config.
	// coffee-claws-middle: gripper-attached collision box (parent gripper).
	claws, err := LoadToolFrame(configPath, "coffee-claws-middle")
	if err != nil {
		return err
	}
	if err := addToolFrame(fs, claws, gripper); err != nil {
		return err
	}

	// filter: sphere tool tip (parent gripper) — the planning target frame.
	filter, err := LoadToolFrame(configPath, "filter")
	if err != nil {
		return err
	}
	filterFrame, err := frameFor(filter)
	if err != nil {
		return err
	}
	if err := fs.AddFrame(filterFrame, gripper); err != nil {
		return fmt.Errorf("add filter frame: %w", err)
	}

	// portafilter-handle: box hanging below the filter (parent filter).
	handle, err := LoadToolFrame(configPath, "portafilter-handle")
	if err != nil {
		return err
	}
	if err := addToolFrame(fs, handle, filterFrame); err != nil {
		return err
	}
	return nil
}

// addToolFrame builds a static frame with geometry for a tool-chain obstacle and
// attaches it under parent.
func addToolFrame(fs *referenceframe.FrameSystem, o Obstacle, parent referenceframe.Frame) error {
	f, err := frameFor(o)
	if err != nil {
		return err
	}
	if err := fs.AddFrame(f, parent); err != nil {
		return fmt.Errorf("add %s frame: %w", o.Name, err)
	}
	return nil
}

// addObstacles attaches every world-scene obstacle from the config in an order
// that guarantees each frame's parent is already present.
func addObstacles(fs *referenceframe.FrameSystem, configPath string) error {
	obstacles, err := LoadObstacles(configPath)
	if err != nil {
		return fmt.Errorf("load obstacles: %w", err)
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
			f, err := frameFor(o)
			if err != nil {
				return err
			}
			if err := fs.AddFrame(f, parent); err != nil {
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
func frameFor(o Obstacle) (referenceframe.Frame, error) {
	pose := spatialmath.NewPose(o.Translation, o.Orientation)
	f, err := referenceframe.NewStaticFrameWithGeometry(o.Name, pose, o.Geometry)
	if err != nil {
		return nil, fmt.Errorf("create %s frame: %w", o.Name, err)
	}
	return f, nil
}
