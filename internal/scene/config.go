// Package scene parses the Beanjamin machine config into the obstacle frames
// and planning frame system the baker uses to plan the brew sequence.
package scene

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/golang/geo/r3"
	"go.viam.com/rdk/spatialmath"
)

// obstacleModel is the Viam component model used for every obstacle frame in
// the Beanjamin config.
const obstacleModel = "erh:vmodutils:obstacle"

// toolFrameNames is the single source of truth for the arm's tool-chain frames.
// The frame-system builder mounts these on the arm (see addToolChain), and
// LoadObstacles excludes any obstacle parented to one of them so the world scene
// and the tool chain can't drift apart. Keep this in sync with the frames built
// in addToolChain.
var toolFrameNames = map[string]bool{
	"gripper":             true,
	"filter":              true,
	"portafilter-handle":  true,
	"coffee-claws-middle": true,
}

// Obstacle is a single static obstacle frame from the machine config, parented
// in the world scene (never on the arm's tool chain).
type Obstacle struct {
	Name        string
	Parent      string
	Translation r3.Vector
	Orientation spatialmath.Orientation
	Geometry    spatialmath.Geometry
}

// LoadObstacles reads the machine config at configPath and returns the
// world-scene obstacle frames keyed by name.
//
// Only components whose model is the obstacle model and whose frame carries a
// geometry are considered. Gripper-attached frames (parented to "gripper" or
// "filter" — the tool chain modeled in the frame-system builder) are excluded.
// Frames whose translation, orientation, or geometry is an unresolved config
// $variable are skipped: this merged config does not bind every machine-specific
// variable, so such frames cannot be placed and are left for a fully-resolved
// config export.
func LoadObstacles(configPath string) (map[string]Obstacle, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", configPath, err)
	}

	var cfg config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", configPath, err)
	}

	vars := cfg.variables()

	obstacles := make(map[string]Obstacle)
	for _, comp := range cfg.Components {
		if comp.Model != obstacleModel || comp.Frame == nil {
			continue
		}
		if comp.Disabled {
			continue // disabled on the real machine (e.g. wall) — not a real obstacle
		}
		fr := comp.Frame
		if toolFrameNames[fr.Parent] {
			continue // gripper-attached tool chain, handled elsewhere
		}

		geom, ok, err := resolveGeometry(fr.Geometry, vars, comp.Name)
		if err != nil {
			return nil, fmt.Errorf("obstacle %q: %w", comp.Name, err)
		}
		if !ok {
			continue // no geometry on this frame
		}
		translation, ok, err := resolveVector(fr.Translation, vars, comp.Name)
		if err != nil {
			return nil, fmt.Errorf("obstacle %q: %w", comp.Name, err)
		}
		if !ok {
			continue // no translation on this frame
		}
		orientation, ok, err := resolveOrientation(fr.Orientation, vars, comp.Name)
		if err != nil {
			return nil, fmt.Errorf("obstacle %q: %w", comp.Name, err)
		}
		if !ok {
			continue // no orientation on this frame
		}

		obstacles[comp.Name] = Obstacle{
			Name:        comp.Name,
			Parent:      fr.Parent,
			Translation: translation,
			Orientation: orientation,
			Geometry:    geom,
		}
	}

	return obstacles, nil
}

// LoadToolFrame reads a single named component from the machine config and
// returns it as an Obstacle (frame transform + geometry). Unlike LoadObstacles
// it does not filter by parent, so it can read the gripper-attached tool-chain
// frames (filter, portafilter-handle, coffee-claws-middle) that the frame-system
// builder mounts on the arm. It errors if the component is missing or has no
// resolvable frame/geometry.
func LoadToolFrame(configPath, name string) (Obstacle, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return Obstacle{}, fmt.Errorf("reading config %s: %w", configPath, err)
	}

	var cfg config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Obstacle{}, fmt.Errorf("parsing config %s: %w", configPath, err)
	}

	vars := cfg.variables()
	for _, comp := range cfg.Components {
		if comp.Name != name {
			continue
		}
		if comp.Frame == nil {
			return Obstacle{}, fmt.Errorf("tool frame %q has no frame", name)
		}
		fr := comp.Frame
		geom, ok, err := resolveGeometry(fr.Geometry, vars, comp.Name)
		if err != nil {
			return Obstacle{}, fmt.Errorf("tool frame %q: %w", name, err)
		}
		if !ok {
			return Obstacle{}, fmt.Errorf("tool frame %q has no resolvable geometry", name)
		}
		translation, ok, err := resolveVector(fr.Translation, vars, comp.Name)
		if err != nil {
			return Obstacle{}, fmt.Errorf("tool frame %q: %w", name, err)
		}
		if !ok {
			return Obstacle{}, fmt.Errorf("tool frame %q has no resolvable translation", name)
		}
		orientation, ok, err := resolveOrientation(fr.Orientation, vars, comp.Name)
		if err != nil {
			return Obstacle{}, fmt.Errorf("tool frame %q: %w", name, err)
		}
		if !ok {
			return Obstacle{}, fmt.Errorf("tool frame %q has no resolvable orientation", name)
		}
		return Obstacle{
			Name:        comp.Name,
			Parent:      fr.Parent,
			Translation: translation,
			Orientation: orientation,
			Geometry:    geom,
		}, nil
	}
	return Obstacle{}, fmt.Errorf("tool frame %q not found in config", name)
}

// config is the subset of the machine config the baker reads.
type config struct {
	Components []component        `json:"components"`
	Fragments  []fragmentWithVars `json:"fragments"`
}

// variables merges every fragment's variable bindings into one resolution map.
func (c config) variables() map[string]json.RawMessage {
	out := make(map[string]json.RawMessage)
	for _, f := range c.Fragments {
		for name, value := range f.Variables {
			out[name] = value
		}
	}
	return out
}

type fragmentWithVars struct {
	Variables map[string]json.RawMessage `json:"variables"`
}

type component struct {
	Name     string `json:"name"`
	Model    string `json:"model"`
	Frame    *frame `json:"frame"`
	Disabled bool   `json:"disabled"`
}

type frame struct {
	Parent      string          `json:"parent"`
	Translation json.RawMessage `json:"translation"`
	Orientation json.RawMessage `json:"orientation"`
	Geometry    json.RawMessage `json:"geometry"`
}

// vector is a {x,y,z} translation in millimeters.
type vector struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// orientationValue is the {x,y,z,th} ov-degrees payload of a frame orientation.
type orientationValue struct {
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
	Z  float64 `json:"z"`
	Th float64 `json:"th"`
}

// geometry is a frame geometry: box dims (x,y,z) or sphere radius (r).
type geometry struct {
	Type string  `json:"type"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Z    float64 `json:"z"`
	R    float64 `json:"r"`
}

// resolveRaw returns the concrete JSON for a frame field, following a
// {"$variable": {"name": ...}} reference into the variables map.
//
// A nil result with a nil error means the field is absent (caller may legitimately
// skip it). When the field references a $variable that has no binding in this
// merged config it returns an error rather than nil/false: a silently-dropped
// reference would mean a missing collision volume, so unresolved variables must
// fail loudly. This branch guards future config re-exports; the committed config
// is fully resolved and should not trigger it.
func resolveRaw(raw json.RawMessage, vars map[string]json.RawMessage, label string) (json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var ref struct {
		Variable *struct {
			Name string `json:"name"`
		} `json:"$variable"`
	}
	if err := json.Unmarshal(raw, &ref); err == nil && ref.Variable != nil {
		bound, ok := vars[ref.Variable.Name]
		if !ok {
			return nil, fmt.Errorf("unresolved $variable %q referenced by %q", ref.Variable.Name, label)
		}
		return bound, nil
	}
	return raw, nil
}

func resolveVector(raw json.RawMessage, vars map[string]json.RawMessage, label string) (r3.Vector, bool, error) {
	resolved, err := resolveRaw(raw, vars, label)
	if err != nil {
		return r3.Vector{}, false, err
	}
	if resolved == nil {
		return r3.Vector{}, false, nil
	}
	var v vector
	if err := json.Unmarshal(resolved, &v); err != nil {
		return r3.Vector{}, false, nil
	}
	return r3.Vector{X: v.X, Y: v.Y, Z: v.Z}, true, nil
}

func resolveOrientation(raw json.RawMessage, vars map[string]json.RawMessage, label string) (spatialmath.Orientation, bool, error) {
	resolved, err := resolveRaw(raw, vars, label)
	if err != nil {
		return nil, false, err
	}
	if resolved == nil {
		return nil, false, nil
	}
	var o struct {
		Type  string           `json:"type"`
		Value orientationValue `json:"value"`
	}
	if err := json.Unmarshal(resolved, &o); err != nil {
		return nil, false, nil
	}
	// Only ov_degrees is understood. An empty type is treated as ov_degrees
	// (the config default); any other type would be silently misinterpreted as
	// OrientationVectorDegrees, so reject it loudly.
	if o.Type != "" && o.Type != "ov_degrees" {
		return nil, false, fmt.Errorf("unsupported orientation type %q for %q", o.Type, label)
	}
	return &spatialmath.OrientationVectorDegrees{
		OX:    o.Value.X,
		OY:    o.Value.Y,
		OZ:    o.Value.Z,
		Theta: o.Value.Th,
	}, true, nil
}

func resolveGeometry(raw json.RawMessage, vars map[string]json.RawMessage, label string) (spatialmath.Geometry, bool, error) {
	resolved, err := resolveRaw(raw, vars, label)
	if err != nil {
		return nil, false, err
	}
	if resolved == nil {
		return nil, false, nil
	}
	var g geometry
	if err := json.Unmarshal(resolved, &g); err != nil {
		return nil, false, nil
	}

	switch g.Type {
	case "box":
		geom, err := spatialmath.NewBox(spatialmath.NewZeroPose(), r3.Vector{X: g.X, Y: g.Y, Z: g.Z}, label)
		if err != nil {
			return nil, false, nil
		}
		return geom, true, nil
	case "sphere":
		geom, err := spatialmath.NewSphere(spatialmath.NewZeroPose(), g.R, label)
		if err != nil {
			return nil, false, nil
		}
		return geom, true, nil
	default:
		return nil, false, nil
	}
}
