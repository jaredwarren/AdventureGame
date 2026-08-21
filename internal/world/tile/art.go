package tile

import (
	"fmt"
	"strings"
)

// SpatialMode selects how GridPos picks among Spatial.Variants.
const SpatialModeGridHash = "gridHash"

// Art is the editable vector definition for one tile GID.
type Art struct {
	GID     int           `json:"gid"`
	Name    string        `json:"name"`
	Size    int           `json:"size"`
	Layers  []Shape       `json:"layers"`
	Spatial *SpatialGroup `json:"spatial,omitempty"`
}

// SpatialGroup holds weighted micro-variants drawn on top of Layers.
type SpatialGroup struct {
	Mode     string           `json:"mode"`
	Variants []SpatialVariant `json:"variants"`
}

// SpatialVariant is one weighted look selected by gridHash.
type SpatialVariant struct {
	ID     string  `json:"id"`
	Weight int     `json:"weight"`
	Layers []Shape `json:"layers"`
}

// Shape is one editable vector primitive in tile-local coordinates (0..Size).
type Shape struct {
	ID   string `json:"id"`
	Type string `json:"type"` // rect | line | circle | path
	// Coords must not use omitempty: x=0,y=0 is a full-tile fill origin, and
	// dropping zeros makes the browser treat them as undefined → NaN → no draw.
	X           float32   `json:"x"`
	Y           float32   `json:"y"`
	W           float32   `json:"w"`
	H           float32   `json:"h"`
	X1          float32   `json:"x1"`
	Y1          float32   `json:"y1"`
	X2          float32   `json:"x2"`
	Y2          float32   `json:"y2"`
	CX          float32   `json:"cx"`
	CY          float32   `json:"cy"`
	R           float32   `json:"r"`
	Fill        string    `json:"fill,omitempty"`
	Stroke      string    `json:"stroke,omitempty"`
	StrokeWidth float32   `json:"strokeWidth,omitempty"`
	Filled      *bool     `json:"filled,omitempty"` // path only; default true when Fill set
	Segs        []PathSeg `json:"segs,omitempty"`
}

// PathSeg is one path command: M (move), L (line), Q (quad), Z (close).
type PathSeg struct {
	Op string  `json:"op"` // M | L | Q | Z
	X  float32 `json:"x"`
	Y  float32 `json:"y"`
	CX float32 `json:"cx"` // quadratic control
	CY float32 `json:"cy"`
}

// Validate checks structural constraints for editor saves and runtime load.
func (a Art) Validate() error {
	if a.GID < 0 {
		return fmt.Errorf("gid must be >= 0")
	}
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if a.Size <= 0 {
		return fmt.Errorf("size must be > 0")
	}
	for i, s := range a.Layers {
		if err := s.validate(fmt.Sprintf("layers[%d]", i)); err != nil {
			return err
		}
	}
	if a.Spatial == nil {
		return nil
	}
	if a.Spatial.Mode != "" && a.Spatial.Mode != SpatialModeGridHash {
		return fmt.Errorf("spatial.mode %q is unsupported (want %q)", a.Spatial.Mode, SpatialModeGridHash)
	}
	if len(a.Spatial.Variants) == 0 {
		return fmt.Errorf("spatial.variants must be non-empty when spatial is set")
	}
	total := 0
	ids := make(map[string]bool, len(a.Spatial.Variants))
	for i, v := range a.Spatial.Variants {
		if strings.TrimSpace(v.ID) == "" {
			return fmt.Errorf("spatial.variants[%d].id is required", i)
		}
		if ids[v.ID] {
			return fmt.Errorf("spatial.variants[%d].id %q is duplicated", i, v.ID)
		}
		ids[v.ID] = true
		if v.Weight <= 0 {
			return fmt.Errorf("spatial.variants[%d].weight must be > 0", i)
		}
		total += v.Weight
		for j, s := range v.Layers {
			if err := s.validate(fmt.Sprintf("spatial.variants[%d].layers[%d]", i, j)); err != nil {
				return err
			}
		}
	}
	if total != 100 {
		return fmt.Errorf("spatial variant weights must sum to 100, got %d", total)
	}
	return nil
}

func (s Shape) validate(path string) error {
	if strings.TrimSpace(s.ID) == "" {
		return fmt.Errorf("%s.id is required", path)
	}
	switch s.Type {
	case "rect":
		if s.W < 0 || s.H < 0 {
			return fmt.Errorf("%s: rect w/h must be >= 0", path)
		}
		if s.Fill == "" && s.Stroke == "" {
			return fmt.Errorf("%s: rect needs fill or stroke", path)
		}
	case "line":
		if s.Stroke == "" {
			return fmt.Errorf("%s: line needs stroke", path)
		}
	case "circle":
		if s.R < 0 {
			return fmt.Errorf("%s: circle r must be >= 0", path)
		}
		if s.Fill == "" && s.Stroke == "" {
			return fmt.Errorf("%s: circle needs fill or stroke", path)
		}
	case "path":
		if len(s.Segs) == 0 {
			return fmt.Errorf("%s: path needs segs", path)
		}
		if s.Fill == "" && s.Stroke == "" {
			return fmt.Errorf("%s: path needs fill or stroke", path)
		}
		for i, seg := range s.Segs {
			switch seg.Op {
			case "M", "L", "Q", "Z":
			default:
				return fmt.Errorf("%s.segs[%d].op %q is invalid", path, i, seg.Op)
			}
		}
	default:
		return fmt.Errorf("%s.type %q is invalid", path, s.Type)
	}
	if s.Fill != "" {
		if _, err := ParseHexColor(s.Fill); err != nil {
			return fmt.Errorf("%s.fill: %w", path, err)
		}
	}
	if s.Stroke != "" {
		if _, err := ParseHexColor(s.Stroke); err != nil {
			return fmt.Errorf("%s.stroke: %w", path, err)
		}
	}
	return nil
}
