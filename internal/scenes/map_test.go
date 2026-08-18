package scenes

import (
	"image/color"
	"testing"

	"github.com/jaredwarren/game-test/internal/render"
	"github.com/jaredwarren/game-test/internal/run"
	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/world"
)

type mapTestInput struct {
	services.Input
	justPressed map[services.Action]bool
}

func (m *mapTestInput) JustPressed(a services.Action) bool {
	return m.justPressed[a]
}

func (m *mapTestInput) IsDown(a services.Action) bool {
	return false
}

func (m *mapTestInput) IsModifierDown(mod services.Modifier) bool {
	return false
}

func (m *mapTestInput) Axis2D() (int, int) {
	return 0, 0
}

type mapTestRenderer struct {
	services.Renderer
	rects []struct{ x, y, w, h float32 }
	texts []string
}

func (r *mapTestRenderer) Camera() *render.Camera {
	return &render.Camera{}
}

func (r *mapTestRenderer) FillRect(x, y, w, h float32, c color.RGBA) {
	r.rects = append(r.rects, struct{ x, y, w, h float32 }{x, y, w, h})
}

func (r *mapTestRenderer) StrokeRect(x, y, w, h, lw float32, c color.RGBA) {}

func (r *mapTestRenderer) StrokeLine(x1, y1, x2, y2, lw float32, c color.RGBA) {}

func (r *mapTestRenderer) DrawText(x, y int, text string) {
	r.texts = append(r.texts, text)
}

func (r *mapTestRenderer) DrawTextOpt(x, y int, text string, opts services.TextOptions) {
	r.texts = append(r.texts, text)
}

type mapTestContext struct {
	input    services.Input
	renderer services.Renderer
	assets   services.AssetCache
	sess     *run.Session
	mgr      *Manager
}

func (c *mapTestContext) Input() services.Input         { return c.input }
func (c *mapTestContext) Audio() services.Audio         { return nil }
func (c *mapTestContext) Assets() services.AssetCache   { return c.assets }
func (c *mapTestContext) Renderer() services.Renderer   { return c.renderer }
func (c *mapTestContext) Clipboard() services.Clipboard { return nil }
func (c *mapTestContext) Session() *run.Session         { return c.sess }
func (c *mapTestContext) Manager() *Manager             { return c.mgr }

func TestParseAndFormatGridMapID(t *testing.T) {
	cases := []struct {
		id      string
		wantRow int
		wantCol int
		wantOk  bool
	}{
		{"A-1", 0, 0, true},
		{"E-5", 4, 4, true},
		{"J-10", 9, 9, true},
		{"B-7", 1, 6, true},
		{"field1", 0, 0, false},
		{"dun:42", 0, 0, false},
		{"K-1", 0, 0, false},
		{"A-0", 0, 0, false},
		{"A-11", 0, 0, false},
		{"", 0, 0, false},
	}

	for _, tc := range cases {
		r, c, ok := parseGridMapID(tc.id)
		if ok != tc.wantOk {
			t.Errorf("parseGridMapID(%q) ok = %v, want %v", tc.id, ok, tc.wantOk)
		}
		if ok {
			if r != tc.wantRow || c != tc.wantCol {
				t.Errorf("parseGridMapID(%q) = (%d, %d), want (%d, %d)", tc.id, r, c, tc.wantRow, tc.wantCol)
			}
			formatted := formatGridMapID(r, c)
			if formatted != tc.id {
				t.Errorf("formatGridMapID(%d, %d) = %q, want %q", r, c, formatted, tc.id)
			}
		}
	}
}

func TestMapScene_CursorPositionAndNavigation(t *testing.T) {
	sess := run.NewSession()
	sess.World = &world.World{MapID: "D-8"}

	ms := newMapScene().(*MapScene)
	in := &mapTestInput{justPressed: make(map[services.Action]bool)}
	mgr := NewManager()
	ctx := &mapTestContext{input: in, sess: sess, mgr: mgr}

	if err := ms.Enter(ctx, nil); err != nil {
		t.Fatal(err)
	}

	// D-8 corresponds to row 3 (D) and col 7 (8)
	if ms.cursorRow != 3 || ms.cursorCol != 7 {
		t.Fatalf("expected cursor at (3, 7), got (%d, %d)", ms.cursorRow, ms.cursorCol)
	}

	// Move right -> should go to col 8
	in.justPressed[services.ActionMoveRight] = true
	_ = ms.Update(ctx)
	in.justPressed[services.ActionMoveRight] = false
	if ms.cursorCol != 8 {
		t.Errorf("expected col 8, got %d", ms.cursorCol)
	}

	// Move down -> should go to row 4
	in.justPressed[services.ActionMoveDown] = true
	_ = ms.Update(ctx)
	in.justPressed[services.ActionMoveDown] = false
	if ms.cursorRow != 4 {
		t.Errorf("expected row 4, got %d", ms.cursorRow)
	}

	// Move up with wrap-around
	ms.cursorRow = 0
	in.justPressed[services.ActionMoveUp] = true
	_ = ms.Update(ctx)
	in.justPressed[services.ActionMoveUp] = false
	if ms.cursorRow != 9 {
		t.Errorf("expected wrap to row 9, got %d", ms.cursorRow)
	}

	// Move left with wrap-around
	ms.cursorCol = 0
	in.justPressed[services.ActionMoveLeft] = true
	_ = ms.Update(ctx)
	in.justPressed[services.ActionMoveLeft] = false
	if ms.cursorCol != 9 {
		t.Errorf("expected wrap to col 9, got %d", ms.cursorCol)
	}
}

func TestMapScene_ExitOnMapAction(t *testing.T) {
	mgr := NewManager()
	mgr.Register(ScenePlay, newPlayScene)
	mgr.Register(SceneMap, newMapScene)
	mgr.Replace(ScenePlay, nil)

	sess := run.NewSession()
	sess.World = &world.World{MapID: "E-5", HP: 3}
	sess.World.Stats.Vitality = 3
	in := &mapTestInput{justPressed: make(map[services.Action]bool)}
	rend := &mapTestRenderer{}
	ctx := &mapTestContext{input: in, renderer: rend, sess: sess, mgr: mgr}

	if err := mgr.Update(ctx); err != nil {
		t.Fatal(err)
	}

	// Push map overlay
	mgr.PushOverlay(SceneMap, nil)
	if err := mgr.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if !mgr.OverlayActive() {
		t.Fatal("expected SceneMap overlay to be active")
	}

	// Press ActionMap to close
	in.justPressed[services.ActionMap] = true
	if err := mgr.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if mgr.OverlayActive() {
		t.Fatal("expected SceneMap overlay to be closed after ActionMap")
	}
}

func TestMapScene_DrawNoPanic(t *testing.T) {
	sess := run.NewSession()
	sess.World = &world.World{MapID: "E-5", MapW: 20, MapH: 15}
	sess.MarkMapVisited("E-5")
	sess.MarkMapVisited("E-6")

	ms := newMapScene()
	rend := &mapTestRenderer{}
	in := &mapTestInput{justPressed: make(map[services.Action]bool)}
	ctx := &mapTestContext{input: in, renderer: rend, sess: sess}

	_ = ms.Enter(ctx, nil)
	ms.Draw(ctx)

	if len(rend.texts) == 0 {
		t.Error("expected texts to be rendered during MapScene.Draw")
	}
}
