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
)

// EditorScene loads a .tmj from disk, edits ground + markers, previews via
// BuildFromTiled + Renderer.DrawWorld.
type EditorScene struct {
	// mapID is the logical map being edited; resolved to a .tmj path in
	// Enter. Provided by the CLI flag via Manager params.
	mapID string

	path   string
	tm     *tiled.Map
	errMsg string

	modeTile bool // false = marker mode

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

	showTileMenu   bool
	tileMenuSelect int
	tileMenuScroll int

	showItemMenu     bool
	itemMenuSelect   int
	activePickupKind string
}

func newEditorScene() Scene { return &EditorScene{} }

var (
	editorBrushPalette = []int{
		world.GIDGrass, world.GIDWall, world.GIDCracked, world.GIDDoor,
		world.GIDWater, world.GIDLock, world.GIDFloor2, world.GIDTree, world.GIDEmpty,
		world.GIDWaterShoreTop, world.GIDWaterShoreBottom, world.GIDWaterShoreLeft, world.GIDWaterShoreRight,
		world.GIDWaterShoreNE, world.GIDWaterShoreNW, world.GIDWaterShoreSW, world.GIDWaterShoreSE,
		world.GIDWaterShoreNEInner, world.GIDWaterShoreNWInner, world.GIDWaterShoreSWInner, world.GIDWaterShoreSEInner,
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
	s.brushGID = world.GIDGrass
	s.showTileMenu = false
	s.tileMenuSelect = 0
	s.tileMenuScroll = 0
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
