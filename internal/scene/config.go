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
		fr := comp.Frame
		if fr.Parent == "gripper" || fr.Parent == "filter" {
			continue // gripper-attached tool chain, handled elsewhere
		}

		geom, ok := resolveGeometry(fr.Geometry, vars, comp.Name)
		if !ok {
			continue // no geometry, or an unresolved $variable geometry
		}
		translation, ok := resolveVector(fr.Translation, vars)
		if !ok {
			continue // unresolved $variable translation
		}
		orientation, ok := resolveOrientation(fr.Orientation, vars)
		if !ok {
			continue // unresolved $variable orientation
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
		geom, ok := resolveGeometry(fr.Geometry, vars, comp.Name)
		if !ok {
			return Obstacle{}, fmt.Errorf("tool frame %q has no resolvable geometry", name)
		}
		translation, ok := resolveVector(fr.Translation, vars)
		if !ok {
			return Obstacle{}, fmt.Errorf("tool frame %q has no resolvable translation", name)
		}
		orientation, ok := resolveOrientation(fr.Orientation, vars)
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
	Name  string `json:"name"`
	Model string `json:"model"`
	Frame *frame `json:"frame"`
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
// {"$variable": {"name": ...}} reference into the variables map. It reports
// false when the field references a variable that has no binding.
func resolveRaw(raw json.RawMessage, vars map[string]json.RawMessage) (json.RawMessage, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var ref struct {
		Variable *struct {
			Name string `json:"name"`
		} `json:"$variable"`
	}
	if err := json.Unmarshal(raw, &ref); err == nil && ref.Variable != nil {
		bound, ok := vars[ref.Variable.Name]
		return bound, ok
	}
	return raw, true
}

func resolveVector(raw json.RawMessage, vars map[string]json.RawMessage) (r3.Vector, bool) {
	resolved, ok := resolveRaw(raw, vars)
	if !ok {
		return r3.Vector{}, false
	}
	var v vector
	if err := json.Unmarshal(resolved, &v); err != nil {
		return r3.Vector{}, false
	}
	return r3.Vector{X: v.X, Y: v.Y, Z: v.Z}, true
}

func resolveOrientation(raw json.RawMessage, vars map[string]json.RawMessage) (spatialmath.Orientation, bool) {
	resolved, ok := resolveRaw(raw, vars)
	if !ok {
		return nil, false
	}
	var o struct {
		Value orientationValue `json:"value"`
	}
	if err := json.Unmarshal(resolved, &o); err != nil {
		return nil, false
	}
	return &spatialmath.OrientationVectorDegrees{
		OX:    o.Value.X,
		OY:    o.Value.Y,
		OZ:    o.Value.Z,
		Theta: o.Value.Th,
	}, true
}

func resolveGeometry(raw json.RawMessage, vars map[string]json.RawMessage, label string) (spatialmath.Geometry, bool) {
	resolved, ok := resolveRaw(raw, vars)
	if !ok {
		return nil, false
	}
	var g geometry
	if err := json.Unmarshal(resolved, &g); err != nil {
		return nil, false
	}

	switch g.Type {
	case "box":
		geom, err := spatialmath.NewBox(spatialmath.NewZeroPose(), r3.Vector{X: g.X, Y: g.Y, Z: g.Z}, label)
		if err != nil {
			return nil, false
		}
		return geom, true
	case "sphere":
		geom, err := spatialmath.NewSphere(spatialmath.NewZeroPose(), g.R, label)
		if err != nil {
			return nil, false
		}
		return geom, true
	default:
		return nil, false
	}
}
