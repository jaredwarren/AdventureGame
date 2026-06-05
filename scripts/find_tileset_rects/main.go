// One-off: infer tile crops in tileset_alpha.png from a 4×2 grid of 704×768 cells.
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png"

	"github.com/jaredwarren/game-test/assets/sprites"
)

func main() {
	img, _, err := image.Decode(bytes.NewReader(sprites.TilesetAlphaPNG))
	if err != nil {
		panic(err)
	}
	b := img.Bounds()
	cellW := b.Dx() / 4
	cellH := b.Dy() / 2
	fmt.Println("bounds", b.Dx(), b.Dy(), "cell", cellW, cellH)

	isBG := func(c color.Color) bool {
		r, g, bb, a := c.RGBA()
		if a>>8 < 16 {
			return true
		}
		r8, g8, b8 := int(r>>8), int(g>>8), int(bb>>8)
		// dark spec background
		return r8 < 45 && g8 < 45 && b8 < 55
	}

	const inset = 8
	for gid := 0; gid < 8; gid++ {
		col := gid % 4
		row := gid / 4
		xa := b.Min.X + col*cellW
		ya := b.Min.Y + row*cellH
		xb := xa + cellW
		yb := ya + cellH
		minX, minY := xb, yb
		maxX, maxY := xa, ya
		found := false
		for y := ya; y < yb; y++ {
			for x := xa; x < xb; x++ {
				if !isBG(img.At(x, y)) {
					found = true
					if x < minX {
						minX = x
					}
					if x > maxX {
						maxX = x
					}
					if y < minY {
						minY = y
					}
					if y > maxY {
						maxY = y
					}
				}
			}
		}
		if !found {
			fmt.Printf("// gid %d: no content\n", gid)
			continue
		}
		fmt.Printf("image.Rect(%d+%d, %d+%d, %d-%d, %d-%d), // gid %d %dx%d\n",
			minX, inset, minY, inset, maxX+1, inset, maxY+1, inset, gid, maxX-minX+1, maxY-minY+1)
	}
}
