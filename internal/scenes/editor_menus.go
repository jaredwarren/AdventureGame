package scenes

import (
	"image/color"
	"strings"

	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

func (s *EditorScene) drawTileMenu(ctx GameContext) {
	r := ctx.Renderer()
	items := s.currentMenuItems()

	const panelX, panelY float32 = 70, 30
	const panelW, panelH float32 = 180, 180
	const headerH float32 = 22
	const footerH float32 = 18
	const rowH float32 = 20

	// Draw panel background (dark semi-transparent window)
	r.FillRect(panelX, panelY, panelW, panelH, color.RGBA{0x08, 0x0a, 0x14, 0xf0})
	r.StrokeRect(panelX, panelY, panelW, panelH, 1, color.RGBA{0x80, 0x80, 0xa0, 0xff})

	// Header Text
	headerText := "SELECT TILE"
	if s.tileMenuCategory != "" {
		headerText = strings.ToUpper(s.tileMenuCategory)
	}
	r.DrawText(int(panelX)+50, int(panelY)+6, headerText)

	// Divider below header
	r.StrokeLine(panelX+4, panelY+headerH-2, panelX+panelW-4, panelY+headerH-2, 1, color.RGBA{0x50, 0x50, 0x70, 0xff})

	// Scrollable items
	visibleCount := 7
	for i := 0; i < visibleCount; i++ {
		idx := s.tileMenuScroll + i
		if idx >= len(items) {
			break
		}
		item := items[idx]

		rowY := panelY + headerH + float32(i)*rowH

		// Highlight currently selected item in menu
		if idx == s.tileMenuSelect {
			r.FillRect(panelX+4, rowY+1, panelW-8, rowH-2, color.RGBA{0x2b, 0x4a, 0x8a, 0x80})
			r.StrokeRect(panelX+4, rowY+1, panelW-8, rowH-2, 1, color.RGBA{0x50, 0x80, 0xff, 0xff})
		}

		// Draw tile image/thumbnail
		if !item.isBack {
			thumbnailGID := item.gid
			if item.isCategory {
				if fam, ok := tile.FamilyByName(item.category); ok {
					thumbnailGID = fam.BaseGID
				} else {
					thumbnailGID = tile.GIDWall
				}
			}
			r.DrawTileScreen(thumbnailGID, panelX+8, rowY+2, 16, 16)
		} else {
			// Draw simple back arrow or back symbol
			r.DrawText(int(panelX)+10, int(rowY)+4, "<-")
		}

		// Draw tile/item name text
		r.DrawText(int(panelX)+30, int(rowY)+4, item.name)
	}

	// Draw scrollbar
	if len(items) > visibleCount {
		trackX := panelX + panelW - 8
		trackY := panelY + headerH + 2
		trackH := float32(140) - 4
		trackW := float32(4)

		// Track
		r.FillRect(trackX, trackY, trackW, trackH, color.RGBA{0x20, 0x20, 0x30, 0xff})

		// Thumb
		thumbH := trackH * float32(visibleCount) / float32(len(items))
		thumbY := trackY + trackH*float32(s.tileMenuScroll)/float32(len(items))

		r.FillRect(trackX, thumbY, trackW, thumbH, color.RGBA{0x80, 0x80, 0xa0, 0xff})
	}

	// Divider above footer
	r.StrokeLine(panelX+4, panelY+panelH-footerH, panelX+panelW-4, panelY+panelH-footerH, 1, color.RGBA{0x50, 0x50, 0x70, 0xff})

	// Footer hints
	r.DrawText(int(panelX)+10, int(panelY+panelH)-14, "Up/Dn scroll   Enter")
}

func (s *EditorScene) drawItemMenu(ctx GameContext) {
	r := ctx.Renderer()

	const panelX, panelY float32 = 70, 30
	const panelW, panelH float32 = 180, 180
	const headerH float32 = 22
	const footerH float32 = 18
	const rowH float32 = 20

	// Draw panel background (dark semi-transparent window)
	r.FillRect(panelX, panelY, panelW, panelH, color.RGBA{0x08, 0x0a, 0x14, 0xf0})
	r.StrokeRect(panelX, panelY, panelW, panelH, 1, color.RGBA{0x80, 0x80, 0xa0, 0xff})

	// Header Text
	r.DrawText(int(panelX)+50, int(panelY)+6, "SELECT ITEM")

	// Divider below header
	r.StrokeLine(panelX+4, panelY+headerH-2, panelX+panelW-4, panelY+headerH-2, 1, color.RGBA{0x50, 0x50, 0x70, 0xff})

	// Items (no scrolling needed since we only have 5)
	for i, item := range world.AllPickups {
		rowY := panelY + headerH + float32(i)*rowH

		// Highlight currently selected item in menu
		if i == s.itemMenuSelect {
			r.FillRect(panelX+4, rowY+1, panelW-8, rowH-2, color.RGBA{0x2b, 0x4a, 0x8a, 0x80})
			r.StrokeRect(panelX+4, rowY+1, panelW-8, rowH-2, 1, color.RGBA{0x50, 0x80, 0xff, 0xff})
		}

		// Draw pickup image
		r.DrawPickupScreen(item, panelX+8, rowY+2, 16, 16)

		// Draw pickup name/label
		r.DrawText(int(panelX)+30, int(rowY)+4, item.EditorLabel())
	}

	// Divider above footer
	r.StrokeLine(panelX+4, panelY+panelH-footerH, panelX+panelW-4, panelY+panelH-footerH, 1, color.RGBA{0x50, 0x50, 0x70, 0xff})

	// Footer hints
	r.DrawText(int(panelX)+10, int(panelY+panelH)-14, "Up/Dn select   Enter")
}
