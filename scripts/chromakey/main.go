// Chroma-key PNG: pixels matching magenta / pink (high R+B, low G) become transparent.
//
// Usage:
//
//	go run ./scripts/chromakey input.png output.png
//	go run ./scripts/chromakey -near 90 -fringe 3 input.png output.png
//
// Flags:
//
//	-near   max RGB distance from #ff00ff (0 = off)
//	-pink   max RGB distance from typical sheet pink (252,55,248); default 78; 0 = off
//	-fringe neighbor cleanup iterations (removes halos next to transparency); default 2
//	-edge   max RGB distance from pink for fringe pass only; default 118 (looser than -pink)
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

// Typical spec-sheet background (see pickup/player sheets).
const keyR, keyG, keyB = 252, 55, 248

func main() {
	nearFF := flag.Int("near", 0, "if >0, also key pixels within this RGB distance of #ff00ff")
	pinkTol := flag.Float64("pink", 78, "max RGB distance from sheet pink (252,55,248); 0 disables")
	fringeIters := flag.Int("fringe", 2, "passes to eat magenta halos next to transparent pixels (0 = off)")
	edgeTol := flag.Float64("edge", 118, "max distance from pink for fringe pass (only pixels touching transparency)")
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: chromakey [-near N] [-pink D] [-fringe I] [-edge D] input.png output.png")
		os.Exit(2)
	}
	inPath, outPath := args[0], args[1]

	f, err := os.Open(inPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	b := src.Bounds()
	dst := image.NewNRGBA(b)
	dx, dy := b.Dx(), b.Dy()

	for y := 0; y < dy; y++ {
		for x := 0; x < dx; x++ {
			px, py := b.Min.X+x, b.Min.Y+y
			c := src.At(px, py)
			r16, g16, b16, a16 := c.RGBA()
			r8 := uint8(r16 >> 8)
			g8 := uint8(g16 >> 8)
			b8 := uint8(b16 >> 8)
			a8 := uint8(a16 >> 8)

			if keyPass(r8, g8, b8, *nearFF, *pinkTol) {
				dst.SetNRGBA(px, py, color.NRGBA{A: 0})
				continue
			}
			dst.SetNRGBA(px, py, color.NRGBA{R: r8, G: g8, B: b8, A: a8})
		}
	}

	if *fringeIters > 0 && *edgeTol > 0 {
		fringeCleanup(dst, b, *fringeIters, *edgeTol)
	}

	out, err := os.Create(outPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer out.Close()
	if err := png.Encode(out, dst); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func keyPass(r, g, b uint8, nearFF int, pinkTol float64) bool {
	if keyMagentaPink(r, g, b) || keyMagentaLoose(r, g, b) {
		return true
	}
	if nearFF > 0 && distFF00FF(r, g, b) <= float64(nearFF) {
		return true
	}
	if pinkTol > 0 && distKeyPink(r, g, b) <= pinkTol {
		return true
	}
	return false
}

// keyMagentaPink matches strong spec-sheet background.
func keyMagentaPink(r, g, b uint8) bool {
	return r > 230 && b > 230 && g < 100
}

// keyMagentaLoose catches anti-aliased / lighter magenta-pink fringe (still purple-pink, not sprite).
func keyMagentaLoose(r, g, b uint8) bool {
	if r < 195 || b < 195 {
		return false
	}
	if g > 145 {
		return false
	}
	// R and B both well above G (magenta family), even if G crept up on edges
	return int(r)-int(g) > 70 && int(b)-int(g) > 70
}

func distKeyPink(r, g, b uint8) float64 {
	dr := float64(r) - keyR
	dg := float64(g) - keyG
	db := float64(b) - keyB
	return math.Sqrt(dr*dr + dg*dg + db*db)
}

func distFF00FF(r, g, b uint8) float64 {
	const tr, tg, tb = 255, 0, 255
	dr := float64(r) - tr
	dg := float64(g) - tg
	db := float64(b) - tb
	return math.Sqrt(dr*dr + dg*dg + db*db)
}

// fringeCleanup repeatedly makes opaque pixels transparent if they touch transparency
// and still look like residual key color (wider tolerance than the main pass).
func fringeCleanup(img *image.NRGBA, bounds image.Rectangle, iters int, edgeTol float64) {
	dx, dy := bounds.Dx(), bounds.Dy()
	neighDX := []int{-1, 0, 1, -1, 1, -1, 0, 1}
	neighDY := []int{-1, -1, -1, 0, 0, 1, 1, 1}

	for iter := 0; iter < iters; iter++ {
		remove := make([]bool, dx*dy)
		for y := 0; y < dy; y++ {
			for x := 0; x < dx; x++ {
				px, py := bounds.Min.X+x, bounds.Min.Y+y
				c := nrgbaAt(img, px, py)
				if c.A == 0 {
					continue
				}
				touchesClear := false
				for k := range neighDX {
					nx, ny := x+neighDX[k], y+neighDY[k]
					if nx < 0 || ny < 0 || nx >= dx || ny >= dy {
						continue
					}
					nc := nrgbaAt(img, bounds.Min.X+nx, bounds.Min.Y+ny)
					if nc.A == 0 {
						touchesClear = true
						break
					}
				}
				if !touchesClear {
					continue
				}
				if distKeyPink(c.R, c.G, c.B) <= edgeTol ||
					keyMagentaLoose(c.R, c.G, c.B) ||
					fringePurpleEdge(c.R, c.G, c.B) {
					remove[y*dx+x] = true
				}
			}
		}
		changed := false
		for y := 0; y < dy; y++ {
			for x := 0; x < dx; x++ {
				if !remove[y*dx+x] {
					continue
				}
				img.SetNRGBA(bounds.Min.X+x, bounds.Min.Y+y, color.NRGBA{A: 0})
				changed = true
			}
		}
		if !changed {
			break
		}
	}
}

// fringePurpleEdge keys very soft purple halos (R≈B, both moderate, G lower).
func nrgbaAt(img *image.NRGBA, x, y int) color.NRGBA {
	i := img.PixOffset(x, y)
	return color.NRGBA{R: img.Pix[i], G: img.Pix[i+1], B: img.Pix[i+2], A: img.Pix[i+3]}
}

func fringePurpleEdge(r, g, b uint8) bool {
	if g > 165 {
		return false
	}
	rb := (int(r) + int(b)) / 2
	if rb < 175 {
		return false
	}
	d := int(r) - int(b)
	if d < 0 {
		d = -d
	}
	return d < 55 && int(rb)-int(g) > 50
}
