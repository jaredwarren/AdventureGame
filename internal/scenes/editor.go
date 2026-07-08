// editor.go — on-disk .tmj tile/marker editor.
//
// EditorScene reads from disk (not embedded assets) so changes are round-
// trippable. It holds a bit of mutable state (dragging, brush, selection)
// that is strictly scene-local; nothing else in the engine looks at it.
package scenes

import (
	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

// EditorScene loads a .tmj from disk, edits ground + markers, previews via
// BuildFromTiled + Renderer.DrawWorld.
type EditorScene struct {
	mapID string

	path   string
	tm     *tiled.Map
	errMsg string

	modeTile bool // false = marker mode

	activeLayerIndex    int
	showOnlyActiveLayer bool

	brushGID        int
	markerTypeIndex int
	selObj          int // index in markers layer; -1 = none

	dragging   bool
	dragGrabDX float64
	dragGrabDY float64

	painting    bool
	lastPaintTX int
	lastPaintTY int

	savedFlash int

	showTileMenu     bool
	tileMenuSelect   int
	tileMenuScroll   int
	tileMenuCategory string

	showItemMenu     bool
	itemMenuSelect   int
	activePickupKind string
}

func newEditorScene() Scene { return &EditorScene{} }

var (
	editorBrushPalette = []int{
		tile.GIDGrass, tile.GIDWall, tile.GIDCracked, tile.GIDDoor,
		tile.GIDWater, tile.GIDLock, tile.GIDFloor2, tile.GIDTree, tile.GIDEmpty,
		tile.GIDWaterShoreTop, tile.GIDWaterShoreBottom, tile.GIDWaterShoreLeft, tile.GIDWaterShoreRight,
		tile.GIDWaterShoreNE, tile.GIDWaterShoreNW, tile.GIDWaterShoreSW, tile.GIDWaterShoreSE,
		tile.GIDWaterShoreNEInner, tile.GIDWaterShoreNWInner, tile.GIDWaterShoreSWInner, tile.GIDWaterShoreSEInner,
		tile.GIDWallTop, tile.GIDWallBottom, tile.GIDWallLeft, tile.GIDWallRight,
		tile.GIDWallNE, tile.GIDWallNW, tile.GIDWallSW, tile.GIDWallSE,
		tile.GIDWallNEInner, tile.GIDWallNWInner, tile.GIDWallSWInner, tile.GIDWallSEInner,
		tile.GIDRock, tile.GIDRockTop, tile.GIDRockBottom, tile.GIDRockLeft, tile.GIDRockRight,
		tile.GIDRockNE, tile.GIDRockNW, tile.GIDRockSW, tile.GIDRockSE,
		tile.GIDRockNEInner, tile.GIDRockNWInner, tile.GIDRockSWInner, tile.GIDRockSEInner,
		tile.GIDQuicksand, tile.GIDMud, tile.GIDIce, tile.GIDLava, tile.GIDSign,
	}
	// editorBrushActions aligns 1:1 with editorBrushPalette; index N selects palette[N].
	editorBrushActions = []services.Action{
		services.ActionEditorBrush1, services.ActionEditorBrush2,
		services.ActionEditorBrush3, services.ActionEditorBrush4,
		services.ActionEditorBrush5, services.ActionEditorBrush6,
		services.ActionEditorBrush7, services.ActionEditorBrush8,
	}
)

func (s *EditorScene) ID() SceneID { return SceneEditor }

func (s *EditorScene) Enter(ctx GameContext, params map[string]any) error {
	if params != nil {
		if v, ok := params["mapID"].(string); ok {
			s.mapID = v
		}
	}
	s.selObj = -1
	s.brushGID = tile.GIDGrass
	s.showTileMenu = false
	s.tileMenuSelect = 0
	s.tileMenuScroll = 0
	s.tileMenuCategory = ""
	s.showItemMenu = false
	s.itemMenuSelect = 0
	if pickups := world.AllPickups; len(pickups) > 0 {
		s.activePickupKind = pickups[0].TiledName()
	}
	return nil
}

func (s *EditorScene) Exit(ctx GameContext) error {
	ctx.Session().World = nil
	return nil
}
