package ebitenplat

import (
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

type drawItem struct {
	footY float64
	draw  func()
}

// DrawWorld renders the tilemap + entities in camera space with viewport clamping and Y-sorting.
func (r *Renderer) DrawWorld(w *world.World) {
	if w == nil || r.screen == nil {
		return
	}
	screen := r.screen
	ox, oy := r.camOffX, r.camOffY
	vp := screen.Bounds()

	// 1. Viewport-clamped ground tile loop
	const tileSize = float64(tile.Size)
	tx0 := int(math.Floor(ox / tileSize))
	ty0 := int(math.Floor(oy / tileSize))
	tx1 := int(math.Ceil((ox + float64(vp.Dx())) / tileSize))
	ty1 := int(math.Ceil((oy + float64(vp.Dy())) / tileSize))

	if tx0 < 0 {
		tx0 = 0
	}
	if ty0 < 0 {
		ty0 = 0
	}
	if tx1 > w.MapW {
		tx1 = w.MapW
	}
	if ty1 > w.MapH {
		ty1 = w.MapH
	}

	for ty := ty0; ty < ty1; ty++ {
		for tx := tx0; tx < tx1; tx++ {
			idx := ty*w.MapW + tx
			x := float32(float64(tx*tile.Size) - ox)
			y := float32(float64(ty*tile.Size) - oy)

			if len(w.Layers) > 0 {
				for k := 0; k < len(w.Layers); k++ {
					if w.ActiveLayerFilter >= 0 && k != w.ActiveLayerFilter {
						continue
					}
					if idx >= len(w.Layers[k]) {
						continue
					}
					gid := w.Layers[k][idx]
					if gid == tile.GIDEmpty {
						continue
					}
					if w.DestroyedTiles[idx] {
						if def := world.TileDefOf(gid); def.Destroyable() || def.OpenableByKey {
							continue
						}
					}
					if useTileSprites {
						r.drawTile(gid, x, y)
					} else {
						r.drawVectorTile(screen, gid, x, y, tile.Size, tile.Size)
					}
				}
			} else {
				gid := w.Tiles[idx]
				if w.DestroyedTiles[idx] {
					if def := world.TileDefOf(gid); def.Destroyable() {
						gid = def.ResolvedDestroyedGID()
					}
				}
				if useTileSprites {
					r.drawTile(gid, x, y)
				} else {
					r.drawVectorTile(screen, gid, x, y, tile.Size, tile.Size)
				}
			}
		}
	}

	// 2. Build Y-sorted draw list for entities and props
	r.drawList = r.drawList[:0]

	for _, p := range w.Pickups {
		if p.Gone {
			continue
		}
		pickup := p
		footY := pickup.Y + pickup.H
		r.drawList = append(r.drawList, drawItem{
			footY: footY,
			draw: func() {
				if w.IsEditor || pickup.PersistentSaveKey == "" {
					r.drawPickup(pickup, ox, oy)
				} else {
					if pickup.Opened {
						r.drawOpenedChest(pickup, ox, oy)
					} else {
						r.drawClosedChest(pickup, ox, oy)
					}
				}
			},
		})
	}

	for _, e := range w.Enemies {
		if e.HP <= 0 {
			continue
		}
		enemy := e
		footY := enemy.Y + float64(enemy.Hitbox.H)
		r.drawList = append(r.drawList, drawItem{
			footY: footY,
			draw: func() {
				r.drawCharacter(float32(enemy.X-ox), float32(enemy.Y-oy),
					float32(enemy.Hitbox.W), float32(enemy.Hitbox.H),
					enemy.Dir, false, enemy.IsBoss)
			},
		})
	}

	for _, sh := range w.Shrines {
		shrine := sh
		footY := shrine.Rect.Y + shrine.Rect.H
		r.drawList = append(r.drawList, drawItem{
			footY: footY,
			draw: func() {
				r.drawShrine(shrine, ox, oy, w.Tick)
			},
		})
	}

	for _, b := range w.ActiveBombs {
		bomb := b
		footY := bomb.Y
		r.drawList = append(r.drawList, drawItem{
			footY: footY,
			draw: func() {
				r.drawBomb(bomb, ox, oy)
			},
		})
	}

	for _, f := range w.Flames {
		flame := f
		footY := flame.Y
		r.drawList = append(r.drawList, drawItem{
			footY: footY,
			draw: func() {
				r.drawFlame(flame, ox, oy, w.Tick)
			},
		})
	}

	pr := w.PlayerRect()
	playerFootY := pr.Y + pr.H
	r.drawList = append(r.drawList, drawItem{
		footY: playerFootY,
		draw: func() {
			r.drawCharacter(float32(pr.X-ox), float32(pr.Y-oy), float32(pr.W), float32(pr.H), w.Player.Dir, true, false)
			r.drawSwordSwing(w, pr.X, pr.Y, pr.W, pr.H, ox, oy)
			r.drawTorchSwing(w, pr.X, pr.Y, pr.W, pr.H, ox, oy)
		},
	})

	// 3. Sort draw items by footY ascending
	sort.Slice(r.drawList, func(i, j int) bool {
		return r.drawList[i].footY < r.drawList[j].footY
	})

	// 4. Draw all sorted items
	for i := range r.drawList {
		r.drawList[i].draw()
	}

	// 5. Draw editor door overlays
	if w.IsEditor {
		for _, d := range w.Doors {
			rd := d.Rect
			vector.FillRect(screen,
				float32(rd.X-ox), float32(rd.Y-oy), float32(rd.W), float32(rd.H),
				color.RGBA{0xe0, 0x90, 0x18, 0x55}, false)
			vector.StrokeRect(screen,
				float32(rd.X-ox), float32(rd.Y-oy), float32(rd.W), float32(rd.H), 2,
				color.RGBA{0xff, 0xc8, 0x40, 0xff}, false)
		}
	}

	// 6. Apply night lighting overlay
	r.drawNightOverlay(w)
}
