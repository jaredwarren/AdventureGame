// map.go — interactive overworld map overlay.
//
// Shows a 10x10 grid of the world (A-1 through J-10):
//   - Explored / visited cells render as pixel-accurate 20x15 tile miniatures.
//   - Unvisited cells render as black boxes with their letter-number grid coordinates.
//   - The player's active position is indicated with a pulsing marker.
//   - Discovered/activated shrines are highlighted.
//   - The user can move a cursor around the grid using directional controls.
package scenes

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

const (
	gridCols   = 10
	gridRows   = 10
	cellW      = 20 // 20 tiles * 1 px/tile
	cellH      = 15 // 15 tiles * 1 px/tile
	mapOriginX = 60
	mapOriginY = 44
)

// MapScene provides the overworld map popup overlay.
type MapScene struct {
	cursorRow int // 0..9 (A..J)
	cursorCol int // 0..9 (1..10)
	tileCache map[string][]int
	animTick  int
}

func newMapScene() Scene {
	return &MapScene{
		cursorRow: 4, // Default to E (row 4)
		cursorCol: 4, // Default to 5 (col 4)
		tileCache: make(map[string][]int),
	}
}

func (s *MapScene) ID() SceneID { return SceneMap }

func (s *MapScene) Enter(ctx GameContext, params map[string]any) error {
	s.animTick = 0
	sess := ctx.Session()
	if sess != nil && sess.World != nil {
		row, col, ok := parseGridMapID(sess.World.MapID)
		if ok {
			s.cursorRow = row
			s.cursorCol = col
		}
	}
	return nil
}

func (s *MapScene) Exit(ctx GameContext) error { return nil }

func (s *MapScene) Update(ctx GameContext) error {
	s.animTick++
	in := ctx.Input()
	sess := ctx.Session()

	HandleSessionDebugInput(sess, in)

	if in.JustPressed(services.ActionMap) ||
		in.JustPressed(services.ActionCancel) ||
		in.JustPressed(services.ActionPause) ||
		in.JustPressed(services.ActionConfirm) {
		ctx.Manager().PopOverlay()
		return nil
	}

	moved := false
	if in.JustPressed(services.ActionMoveUp) {
		s.cursorRow = (s.cursorRow - 1 + gridRows) % gridRows
		moved = true
	}
	if in.JustPressed(services.ActionMoveDown) {
		s.cursorRow = (s.cursorRow + 1) % gridRows
		moved = true
	}
	if in.JustPressed(services.ActionMoveLeft) {
		s.cursorCol = (s.cursorCol - 1 + gridCols) % gridCols
		moved = true
	}
	if in.JustPressed(services.ActionMoveRight) {
		s.cursorCol = (s.cursorCol + 1) % gridCols
		moved = true
	}

	if moved && ctx.Audio() != nil {
		ctx.Audio().Play("pickup.wav", 0.1)
	}

	return nil
}

func (s *MapScene) Draw(ctx GameContext) {
	sess := ctx.Session()
	r := ctx.Renderer()
	assets := ctx.Assets()

	DrawOverlayDim(r)

	// Outer card backdrop
	const cardX, cardY float32 = 24, 10
	const cardW, cardH float32 = 272, 220
	r.FillRect(cardX, cardY, cardW, cardH, color.RGBA{0x0a, 0x0e, 0x1a, 0xf0})
	r.StrokeRect(cardX, cardY, cardW, cardH, 1, color.RGBA{0x50, 0x60, 0x85, 0xff})

	// Header: Title and current map location
	r.DrawTextOpt(108, 16, "OVERWORLD MAP", services.TextOptions{
		Color: color.RGBA{0xff, 0xd7, 0x00, 0xff},
	})

	currentMap := "None"
	playerRow, playerCol := -1, -1
	isOverworld := false
	if sess != nil && sess.World != nil {
		currentMap = sess.World.MapID
		if rIdx, cIdx, ok := parseGridMapID(currentMap); ok {
			playerRow = rIdx
			playerCol = cIdx
			isOverworld = true
		}
	}

	locText := fmt.Sprintf("Location: %s", currentMap)
	r.DrawText(36, 28, locText)

	// Draw Column headers (1..10)
	for c := 0; c < gridCols; c++ {
		colLabel := strconv.Itoa(c + 1)
		cx := mapOriginX + c*cellW + (cellW/2 - 3)
		r.DrawTextOpt(cx, mapOriginY-11, colLabel, services.TextOptions{
			Color: color.RGBA{0x80, 0x90, 0xb0, 0xff},
		})
	}

	// Draw Row headers (A..J)
	for row := 0; row < gridRows; row++ {
		rowLabel := string(rune('A' + row))
		ry := mapOriginY + row*cellH + 3
		r.DrawTextOpt(mapOriginX-14, ry, rowLabel, services.TextOptions{
			Color: color.RGBA{0x80, 0x90, 0xb0, 0xff},
		})
	}

	// Draw Map Grid Cells
	for row := 0; row < gridRows; row++ {
		for col := 0; col < gridCols; col++ {
			cellMapID := formatGridMapID(row, col)
			cx := float32(mapOriginX + col*cellW)
			cy := float32(mapOriginY + row*cellH)

			visited := false
			if sess != nil {
				visited = sess.IsMapVisited(cellMapID)
			}
			// Current map is always considered visited
			if isOverworld && row == playerRow && col == playerCol {
				visited = true
			}

			if visited {
				s.drawVisitedCell(r, assets, cellMapID, cx, cy)
			} else {
				s.drawUnvisitedCell(r, row, col, cx, cy)
			}

			// Check for activated shrines in this cell
			if sess != nil {
				prog := sess.ProgressFor(cellMapID)
				if prog != nil && len(prog.ActivatedShrines) > 0 {
					// Draw small glowing gold shrine dot in upper right corner of cell
					r.FillRect(cx+float32(cellW)-4, cy+1, 3, 3, color.RGBA{0xff, 0xd7, 0x00, 0xff})
				}
			}

			// Draw cell separator border
			r.StrokeRect(cx, cy, float32(cellW), float32(cellH), 1, color.RGBA{0x25, 0x2a, 0x3d, 0x99})
		}
	}

	// Draw outer border for the full 10x10 map grid
	r.StrokeRect(float32(mapOriginX), float32(mapOriginY), float32(gridCols*cellW), float32(gridRows*cellH), 1.5, color.RGBA{0x60, 0x70, 0x95, 0xff})

	// Draw Player location marker if player is on the overworld
	if isOverworld && playerRow >= 0 && playerCol >= 0 {
		px := float32(mapOriginX + playerCol*cellW)
		py := float32(mapOriginY + playerRow*cellH)

		// Calculate player position relative to cell if world exists
		if sess.World != nil && sess.World.MapW > 0 && sess.World.MapH > 0 {
			relX := float32(sess.World.Player.X / float64(sess.World.MapW*tile.Size))
			relY := float32(sess.World.Player.Y / float64(sess.World.MapH*tile.Size))
			if relX >= 0 && relX <= 1 && relY >= 0 && relY <= 1 {
				markerX := px + relX*float32(cellW) - 1
				markerY := py + relY*float32(cellH) - 1

				// Blink animation every 16 frames
				if (s.animTick/16)%2 == 0 {
					r.FillRect(markerX, markerY, 3, 3, color.RGBA{0xff, 0x30, 0x40, 0xff})
					r.FillRect(markerX+1, markerY+1, 1, 1, color.RGBA{0xff, 0xff, 0xff, 0xff})
				}
			}
		}

		// Also draw subtle green corner brackets around the player's current cell
		if (s.animTick/20)%2 == 0 {
			r.StrokeRect(px, py, float32(cellW), float32(cellH), 1, color.RGBA{0x30, 0xe0, 0x50, 0xaa})
		}
	}

	// Draw Cursor Box around currently selected cell
	curX := float32(mapOriginX + s.cursorCol*cellW)
	curY := float32(mapOriginY + s.cursorRow*cellH)
	r.StrokeRect(curX-1, curY-1, float32(cellW)+2, float32(cellH)+2, 1.5, color.RGBA{0xff, 0xe0, 0x40, 0xff})

	// Status Line below map
	selMapID := formatGridMapID(s.cursorRow, s.cursorCol)
	statusText := "[Unexplored]"
	if sess != nil && (sess.IsMapVisited(selMapID) || (isOverworld && s.cursorRow == playerRow && s.cursorCol == playerCol)) {
		statusText = "[Explored]"
		prog := sess.ProgressFor(selMapID)
		if prog != nil && len(prog.ActivatedShrines) > 0 {
			statusText = "[Explored - Shrine]"
		}
	}
	r.DrawText(36, mapOriginY+gridRows*cellH+6, fmt.Sprintf("Selected: %s %s", selMapID, statusText))

	// Controls Footer
	r.DrawTextOpt(36, mapOriginY+gridRows*cellH+18, "WASD / Arrows: Navigate   M / Esc: Close", services.TextOptions{
		Color: color.RGBA{0x80, 0x90, 0xb0, 0xff},
	})

	if sess != nil && sess.ShowDebugOverlay {
		DrawDebugOverlay(r, sess, s.ID(), DebugOverlayExtras{})
	}
}

// drawVisitedCell renders a miniature 20x15 tilemap for an explored area.
func (s *MapScene) drawVisitedCell(r services.Renderer, assets services.AssetCache, mapID string, x, y float32) {
	tiles := s.getTileData(assets, mapID)
	if len(tiles) < cellW*cellH {
		// Fallback if map data is missing
		r.FillRect(x, y, float32(cellW), float32(cellH), color.RGBA{0x18, 0x30, 0x18, 0xff})
		return
	}

	for ty := 0; ty < cellH; ty++ {
		for tx := 0; tx < cellW; tx++ {
			gid := tiles[ty*cellW+tx]
			if gid == tile.GIDEmpty {
				continue
			}
			sc := tile.DefOf(gid).SwatchColor
			r.FillRect(x+float32(tx), y+float32(ty), 1, 1, sc)
		}
	}
}

// drawUnvisitedCell renders a solid black box with the cell's letter-number label.
func (s *MapScene) drawUnvisitedCell(r services.Renderer, row, col int, x, y float32) {
	r.FillRect(x, y, float32(cellW), float32(cellH), color.RGBA{0x00, 0x00, 0x00, 0xff})

	label := fmt.Sprintf("%c%d", 'A'+row, col+1)
	textX := int(x) + 2
	if len(label) == 3 { // e.g. "A10"
		textX = int(x)
	}
	textY := int(y) + 3

	r.DrawTextOpt(textX, textY, label, services.TextOptions{
		Color: color.RGBA{0x60, 0x68, 0x80, 0xff},
	})
}

// getTileData loads and caches the ground tile GID slice (20x15) for mapID.
func (s *MapScene) getTileData(assets services.AssetCache, mapID string) []int {
	if s.tileCache == nil {
		s.tileCache = make(map[string][]int)
	}
	if data, ok := s.tileCache[mapID]; ok {
		return data
	}

	if assets == nil {
		return nil
	}

	raw, err := assets.MapData(mapID)
	if err != nil {
		return nil
	}

	tm, err := tiled.ParseMap(raw)
	if err != nil {
		return nil
	}

	// Extract ground / main tile layer
	var groundData []int
	for _, layer := range tm.Layers {
		if layer.Type == "tilelayer" && len(layer.Data) >= cellW*cellH {
			groundData = layer.Data
			break
		}
	}

	if len(groundData) > 0 {
		s.tileCache[mapID] = groundData
	}
	return groundData
}

// parseGridMapID extracts (row 0..9, col 0..9) from e.g. "E-5".
func parseGridMapID(id string) (row, col int, ok bool) {
	parts := strings.Split(id, "-")
	if len(parts) != 2 || len(parts[0]) != 1 {
		return 0, 0, false
	}
	rChar := parts[0][0]
	if rChar < 'A' || rChar > 'J' {
		return 0, 0, false
	}
	cNum, err := strconv.Atoi(parts[1])
	if err != nil || cNum < 1 || cNum > gridCols {
		return 0, 0, false
	}
	return int(rChar - 'A'), cNum - 1, true
}

// formatGridMapID formats (row 0..9, col 0..9) into e.g. "E-5".
func formatGridMapID(row, col int) string {
	return fmt.Sprintf("%c-%d", 'A'+row, col+1)
}
