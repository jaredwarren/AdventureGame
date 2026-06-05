package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

type Point struct {
	X, Y int
}

type Rect struct {
	MinX, MinY, MaxX, MaxY int
}

func main() {
	inPath := "assets/sprites/legendofzelda_items_sheet copy.png"
	f, err := os.Open(inPath)
	if err != nil {
		fmt.Printf("Error opening copy: %v\n", err)
		return
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		fmt.Printf("Error decoding copy: %v\n", err)
		return
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// Sprite rects from pickup_atlas.json
	spriteRects := []Rect{
		{70, 70, 70 + 160, 70 + 160},     // Coin
		{680, 245, 680 + 200, 245 + 200}, // Heart
		{90, 560, 90 + 310, 560 + 380},   // Bomb
		{430, 600, 430 + 140, 600 + 320}, // Small Key
		{750, 450, 750 + 150, 450 + 480}, // Torch
	}

	// 1. Run BFS/flood-fill from the image borders to detect the background region.
	bgMask := make([][]bool, h)
	for i := range bgMask {
		bgMask[i] = make([]bool, w)
	}

	var queue []Point

	// Add all border pixels to start BFS
	for x := 0; x < w; x++ {
		for _, y := range []int{0, h - 1} {
			bgMask[y][x] = true
			queue = append(queue, Point{x, y})
		}
	}
	for y := 0; y < h; y++ {
		for _, x := range []int{0, w - 1} {
			if !bgMask[y][x] {
				bgMask[y][x] = true
				queue = append(queue, Point{x, y})
			}
		}
	}

	// A pixel is background-like if R, G, B are all > 120 (since outlines are dark)
	isBackground := func(r, g, b uint8) bool {
		return r > 120 && g > 120 && b > 120
	}

	dx := []int{-1, 1, 0, 0}
	dy := []int{0, 0, -1, 1}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for i := 0; i < 4; i++ {
			nx, ny := curr.X+dx[i], curr.Y+dy[i]
			if nx >= 0 && nx < w && ny >= 0 && ny < h {
				if !bgMask[ny][nx] {
					c := img.At(nx, ny)
					r32, g32, b32, _ := c.RGBA()
					r, g, b := uint8(r32>>8), uint8(g32>>8), uint8(b32>>8)

					if isBackground(r, g, b) {
						bgMask[ny][nx] = true
						queue = append(queue, Point{nx, ny})
					}
				}
			}
		}
	}

	// 2. Build the final image.
	dst := image.NewNRGBA(bounds)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Check if pixel is inside any active sprite rect
			insideActiveSprite := false
			for _, r := range spriteRects {
				if x >= r.MinX && x < r.MaxX && y >= r.MinY && y < r.MaxY {
					insideActiveSprite = true
					break
				}
			}

			if !insideActiveSprite {
				// Rule 1: Outside active sprite bounding boxes -> solid chroma-key green
				dst.SetNRGBA(x, y, color.NRGBA{R: 0, G: 255, B: 0, A: 255})
			} else {
				// Rule 2: Inside active sprite bounding boxes:
				// If it's connected background -> green.
				// Otherwise -> keep original color.
				if bgMask[y][x] {
					dst.SetNRGBA(x, y, color.NRGBA{R: 0, G: 255, B: 0, A: 255})
				} else {
					c := img.At(x, y)
					r32, g32, b32, a32 := c.RGBA()
					dst.SetNRGBA(x, y, color.NRGBA{
						R: uint8(r32 >> 8),
						G: uint8(g32 >> 8),
						B: uint8(b32 >> 8),
						A: uint8(a32 >> 8),
					})
				}
			}
		}
	}

	// 3. Write final image to assets/sprites/legendofzelda_items_sheet.png
	outPath := "assets/sprites/legendofzelda_items_sheet.png"
	out, err := os.Create(outPath)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer out.Close()

	if err := png.Encode(out, dst); err != nil {
		fmt.Printf("Error encoding PNG: %v\n", err)
		return
	}

	fmt.Printf("Successfully chroma-keyed background of %s to solid green (#00FF00)!\n", outPath)
}
