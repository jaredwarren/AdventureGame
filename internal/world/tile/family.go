package tile

import (
	"image/color"
	"sync"

	"github.com/jaredwarren/game-test/internal/geom"
)

// Variant represents one of the 13 standard autotile transition shapes.
type Variant int

const (
	VariantBase Variant = iota
	VariantTop
	VariantBottom
	VariantLeft
	VariantRight
	VariantNE
	VariantNW
	VariantSW
	VariantSE
	VariantNEInner
	VariantNWInner
	VariantSWInner
	VariantSEInner
	VariantCount = 13
)

var variantSuffixes = [VariantCount]string{
	"",
	"_top",
	"_bottom",
	"_left",
	"_right",
	"_ne",
	"_nw",
	"_sw",
	"_se",
	"_ne_inner",
	"_nw_inner",
	"_sw_inner",
	"_se_inner",
}

// FamilyKind declares the collision and surface category for a 13-tile family.
type FamilyKind int

const (
	FamilyFloor FamilyKind = iota // 100% walkable, no collision
	FamilyWall                    // Solid wall barrier with edge collision
	FamilyWater                   // Water surface with edge collision
	FamilyRock                    // Rock wall barrier with edge collision
)

// FamilyConfig declares the full properties for registering a 13-tile transition family.
type FamilyConfig struct {
	Name           string
	Label          string
	Category       string
	BaseGID        int
	ExplicitGIDs   [VariantCount]int
	ExplicitNames  [VariantCount]string
	Kind           FamilyKind
	Surface        SurfaceDef
	DamageKinds    []DamageKind
	Style          TileStyle
	FloorWeight    float64
	Collapsed      bool
	VariantDrawers [VariantCount]func(c Canvas, x, y, w, h float32)
}

// FamilyInfo contains metadata about a registered family for editor menus and palettes.
type FamilyInfo struct {
	Name      string
	Label     string
	Category  string
	BaseGID   int
	GIDs      []int
	Collapsed bool
}

var (
	registryMu         sync.RWMutex
	registeredFamilies []FamilyInfo
	standaloneTiles    []Tile
)

var variantSolidRects = map[Variant][]geom.Rect{
	VariantTop:     {{X: 0, Y: 8, W: 16, H: 8}},
	VariantBottom:  {{X: 0, Y: 0, W: 16, H: 8}},
	VariantLeft:    {{X: 8, Y: 0, W: 8, H: 16}},
	VariantRight:   {{X: 0, Y: 0, W: 8, H: 16}},
	VariantNE:      {{X: 0, Y: 8, W: 8, H: 8}},
	VariantNW:      {{X: 8, Y: 8, W: 8, H: 8}},
	VariantSW:      {{X: 8, Y: 0, W: 8, H: 8}},
	VariantSE:      {{X: 0, Y: 0, W: 8, H: 8}},
	VariantNEInner: {{X: 0, Y: 0, W: 8, H: 16}, {X: 8, Y: 8, W: 8, H: 8}},
	VariantNWInner: {{X: 8, Y: 0, W: 8, H: 16}, {X: 0, Y: 8, W: 8, H: 8}},
	VariantSWInner: {{X: 8, Y: 0, W: 8, H: 16}, {X: 0, Y: 0, W: 8, H: 8}},
	VariantSEInner: {{X: 0, Y: 0, W: 8, H: 16}, {X: 8, Y: 0, W: 8, H: 8}},
}

// RegisterFamily registers all 13 tiles for a transition family.
func RegisterFamily(cfg FamilyConfig) {
	registryMu.Lock()
	defer registryMu.Unlock()

	label := cfg.Label
	if label == "" {
		label = cfg.Name
	}
	category := cfg.Category
	if category == "" {
		category = cfg.Name
	}

	gids := make([]int, VariantCount)
	for i := 0; i < int(VariantCount); i++ {
		v := Variant(i)
		gid := cfg.BaseGID + i
		if cfg.ExplicitGIDs[i] > 0 {
			gid = cfg.ExplicitGIDs[i]
		}
		gids[i] = gid

		name := cfg.Name + variantSuffixes[i]
		if cfg.ExplicitNames[i] != "" {
			name = cfg.ExplicitNames[i]
		}
		var tags []TileTag
		var solidRects []geom.Rect

		switch cfg.Kind {
		case FamilyWall:
			tags = []TileTag{TagSolid, TagWall}
			if v != VariantBase {
				solidRects = variantSolidRects[v]
			}
		case FamilyRock:
			tags = []TileTag{TagSolid, TagWall}
			if v != VariantBase {
				solidRects = variantSolidRects[v]
			}
		case FamilyWater:
			if v == VariantBase {
				tags = []TileTag{TagSolid, TagWater}
			} else {
				tags = []TileTag{TagSolid, TagWater, TagWaterShore}
				solidRects = variantSolidRects[v]
			}
		case FamilyFloor:
			tags = []TileTag{TagFloor}
		}

		drawer := variantDrawer(v, cfg.Style)
		if cfg.VariantDrawers[i] != nil {
			drawer = cfg.VariantDrawers[i]
		}

		swatchColor := cfg.Style.FillColor
		if swatchColor.A == 0 {
			swatchColor = color.RGBA{0x80, 0x80, 0x80, 0xff}
		}

		tileDef := Tile{
			GID:         gid,
			Name:        name,
			Tags:        tags,
			Surface:     cfg.Surface,
			SolidRects:  solidRects,
			DamageKinds: cfg.DamageKinds,
			SwatchColor: swatchColor,
			FloorWeight: cfg.FloorWeight,
			VectorDraw:  drawer,
		}
		defs[gid] = tileDef
	}

	registeredFamilies = append(registeredFamilies, FamilyInfo{
		Name:      cfg.Name,
		Label:     label,
		Category:  category,
		BaseGID:   gids[0],
		GIDs:      gids,
		Collapsed: cfg.Collapsed,
	})
}

func variantDrawer(v Variant, style TileStyle) func(c Canvas, x, y, w, h float32) {
	switch v {
	case VariantBase:
		return makeBaseDrawer(style)
	case VariantTop:
		return makeEdgeDrawer(EdgeTop, style)
	case VariantBottom:
		return makeEdgeDrawer(EdgeBottom, style)
	case VariantLeft:
		return makeEdgeDrawer(EdgeLeft, style)
	case VariantRight:
		return makeEdgeDrawer(EdgeRight, style)
	case VariantNW:
		return makeOuterCornerDrawer(CornerNW, style)
	case VariantNE:
		return makeOuterCornerDrawer(CornerNE, style)
	case VariantSW:
		return makeOuterCornerDrawer(CornerSW, style)
	case VariantSE:
		return makeOuterCornerDrawer(CornerSE, style)
	case VariantNWInner:
		return makeInnerCornerDrawer(CornerNW, style)
	case VariantNEInner:
		return makeInnerCornerDrawer(CornerNE, style)
	case VariantSWInner:
		return makeInnerCornerDrawer(CornerSW, style)
	case VariantSEInner:
		return makeInnerCornerDrawer(CornerSE, style)
	default:
		return makeBaseDrawer(style)
	}
}

// RegisterSingleTile registers a standalone tile in the global tile registry.
func RegisterSingleTile(t Tile) {
	registryMu.Lock()
	defer registryMu.Unlock()
	defs[t.GID] = t
	standaloneTiles = append(standaloneTiles, t)
}

// RegisteredFamilies returns all registered 13-tile autotile families.
func RegisteredFamilies() []FamilyInfo {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]FamilyInfo, len(registeredFamilies))
	copy(out, registeredFamilies)
	return out
}

// FamilyByName returns the registered family metadata matching name.
func FamilyByName(name string) (FamilyInfo, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	for _, f := range registeredFamilies {
		if f.Name == name {
			return f, true
		}
	}
	return FamilyInfo{}, false
}
