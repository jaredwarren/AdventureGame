package ebitenplat

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jaredwarren/game-test/assets/sprites"
	"github.com/jaredwarren/game-test/internal/services"
)

// Image is the Ebiten-backed services.Image impl. Scene draw code can recover
// the underlying *ebiten.Image via Native(); pure systems must not.
type Image struct {
	inner *ebiten.Image
	w, h  int
}

func (e *Image) Size() (int, int)      { return e.w, e.h }
func (e *Image) Native() *ebiten.Image { return e.inner }

// NativeImage extracts the underlying *ebiten.Image from an Image produced by
// this backend. Returns nil if img was not produced here or is a nil frame.
// Only internal/game draw code should call this.
func NativeImage(img services.Image) *ebiten.Image {
	if img == nil {
		return nil
	}
	if n, ok := img.(interface{ Native() *ebiten.Image }); ok {
		return n.Native()
	}
	return nil
}

// AssetCache implements services.AssetCache for the embedded assets currently
// shipped by this project: two atlases (tile, pickup) and Tiled .tmj files.
type AssetCache struct {
	mapFS  embed.FS
	mapDir string // e.g. "maps/"

	mu       sync.Mutex
	atlasMap map[services.AtlasID]services.Atlas
}

// NewAssetCache wires the embed filesystems. mapFS is typically assets.MapFS
// with mapDir "maps/"; atlases are sourced from the sprites package (already
// referenced via //go:embed in assets/sprites/embed.go).
func NewAssetCache(mapFS embed.FS, mapDir string) *AssetCache {
	return &AssetCache{
		mapFS:    mapFS,
		mapDir:   mapDir,
		atlasMap: make(map[services.AtlasID]services.Atlas),
	}
}

// MapData returns the raw .tmj bytes for a map id (e.g. "field1" -> maps/field1.tmj).
func (c *AssetCache) MapData(id string) ([]byte, error) {
	// Try loading from the local assets directory on disk first.
	// This ensures that map edits made in the editor reload immediately in play mode.
	localPath := filepath.Join("assets", "maps", id+".tmj")
	if data, err := os.ReadFile(localPath); err == nil {
		return data, nil
	}
	// Fallback to the embedded maps filesystem.
	return c.mapFS.ReadFile(c.mapDir + id + ".tmj")
}

// Atlas lazily builds and caches the requested atlas. First call for a given
// id performs PNG decode and sub-image slicing; subsequent calls are O(1).
func (c *AssetCache) Atlas(id services.AtlasID) (services.Atlas, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if a, ok := c.atlasMap[id]; ok {
		return a, nil
	}
	var (
		a   services.Atlas
		err error
	)
	switch id {
	case services.AtlasTile:
		a, err = loadTileAtlas()
	case services.AtlasPickup:
		a, err = loadPickupAtlas()
	default:
		return nil, fmt.Errorf("assets: unknown atlas %q", id)
	}
	if err != nil {
		return nil, err
	}
	c.atlasMap[id] = a
	return a, nil
}

// --- concrete atlas impls -------------------------------------------------

type frameAtlas struct {
	frames []services.AtlasFrame
}

func (a *frameAtlas) Frame(i int) services.AtlasFrame {
	if i < 0 || i >= len(a.frames) {
		return services.AtlasFrame{Skip: true}
	}
	return a.frames[i]
}

func (a *frameAtlas) Count() int { return len(a.frames) }

// Tile atlas JSON schema (assets/sprites/tile_atlas.json):
//
//	{ "sprite_sheet": "...", "tile_px": 16, "frames": [{sx,sy,sw,sh,skip?}, ...] }
//
// GIDs map 1:1 to frame indexes (see world.GID* / tile_atlas.json); `skip:true` marks "no art, fallback".
type tileAtlasFile struct {
	SpriteSheet string `json:"sprite_sheet"`
	TilePx      int    `json:"tile_px"`
	Frames      []struct {
		Skip        bool   `json:"skip"`
		SpriteSheet string `json:"sprite_sheet"`
		SX          int    `json:"sx"`
		SY          int    `json:"sy"`
		SW          int    `json:"sw"`
		SH          int    `json:"sh"`
	} `json:"frames"`
}

func loadTileAtlas() (services.Atlas, error) {
	var doc tileAtlasFile
	if err := json.Unmarshal(sprites.TileAtlasJSON, &doc); err != nil {
		return nil, fmt.Errorf("tile atlas: parse json: %w", err)
	}
	if doc.SpriteSheet != sprites.TileSpriteSheetFile {
		return nil, fmt.Errorf("tile atlas: sprite_sheet %q != %q", doc.SpriteSheet, sprites.TileSpriteSheetFile)
	}
	sheet, err := decodeSheet(sprites.OverworldTilePNG)
	if err != nil {
		fmt.Printf("Warning/Error: decode OverworldTilePNG: %v\n", err)
		return nil, fmt.Errorf("tile atlas: %w", err)
	}
	lakeSheet, err := decodeSheet(sprites.LakeIslandScenePNG)
	if err != nil {
		fmt.Printf("Warning/Error: decode LakeIslandScenePNG: %v\n", err)
		return nil, fmt.Errorf("tile atlas lake: %w", err)
	}
	frames := make([]services.AtlasFrame, len(doc.Frames))
	for i, fr := range doc.Frames {
		if fr.Skip || fr.SW < 1 || fr.SH < 1 {
			frames[i] = services.AtlasFrame{Skip: true}
			continue
		}
		currentSheet := sheet
		if fr.SpriteSheet == "lake_island_scene.png" {
			currentSheet = lakeSheet
		}
		b := currentSheet.Bounds()
		r := image.Rect(fr.SX, fr.SY, fr.SX+fr.SW, fr.SY+fr.SH)
		if !r.In(b) {
			frames[i] = services.AtlasFrame{Skip: true}
			continue
		}
		sub, ok := currentSheet.SubImage(r).(*ebiten.Image)
		if !ok {
			frames[i] = services.AtlasFrame{Skip: true}
			continue
		}
		frames[i] = services.AtlasFrame{
			Image: &Image{inner: sub, w: fr.SW, h: fr.SH},
		}
	}
	return &frameAtlas{frames: frames}, nil
}

// Pickup atlas JSON schema (assets/sprites/pickup_atlas.json):
//
//	{ "sprite_sheet": "...", "frames": [{sx,sy,sw,sh,dw,dh,ox,oy}, ...] }
//
// Frame index matches world.PickupKind (coin, heart, bomb, small_key, torch).
type pickupAtlasFile struct {
	SpriteSheet string `json:"sprite_sheet"`
	Frames      []struct {
		SX int     `json:"sx"`
		SY int     `json:"sy"`
		SW int     `json:"sw"`
		SH int     `json:"sh"`
		DW int     `json:"dw"`
		DH int     `json:"dh"`
		OX float64 `json:"ox"`
		OY float64 `json:"oy"`
	} `json:"frames"`
}

// defaultPickupDst mirrors the historical pickupDrawPx constant — atlases that
// omit dw/dh fall back to 12×12 world pixels.
const defaultPickupDst = 12

func loadPickupAtlas() (services.Atlas, error) {
	var doc pickupAtlasFile
	if err := json.Unmarshal(sprites.PickupAtlasJSON, &doc); err != nil {
		return nil, fmt.Errorf("pickup atlas: parse json: %w", err)
	}
	if doc.SpriteSheet != sprites.PickupSpriteSheetFile {
		return nil, fmt.Errorf("pickup atlas: sprite_sheet %q != %q", doc.SpriteSheet, sprites.PickupSpriteSheetFile)
	}
	sheet, err := decodeSheet(sprites.PickupPNG)
	if err != nil {
		return nil, fmt.Errorf("pickup atlas: %w", err)
	}
	b := sheet.Bounds()
	frames := make([]services.AtlasFrame, len(doc.Frames))
	for i, fr := range doc.Frames {
		if fr.SW < 1 || fr.SH < 1 {
			frames[i] = services.AtlasFrame{Skip: true}
			continue
		}
		r := image.Rect(fr.SX, fr.SY, fr.SX+fr.SW, fr.SY+fr.SH)
		if !r.In(b) {
			frames[i] = services.AtlasFrame{Skip: true}
			continue
		}
		sub, ok := sheet.SubImage(r).(*ebiten.Image)
		if !ok {
			frames[i] = services.AtlasFrame{Skip: true}
			continue
		}
		dw, dh := float64(fr.DW), float64(fr.DH)
		if dw < 1 {
			dw = defaultPickupDst
		}
		if dh < 1 {
			dh = defaultPickupDst
		}
		frames[i] = services.AtlasFrame{
			Image:   &Image{inner: sub, w: fr.SW, h: fr.SH},
			DstW:    dw,
			DstH:    dh,
			OffsetX: fr.OX,
			OffsetY: fr.OY,
		}
	}
	return &frameAtlas{frames: frames}, nil
}

func decodeSheet(png []byte) (*ebiten.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(png))
	if err != nil {
		return nil, fmt.Errorf("decode png: %w", err)
	}
	return ebiten.NewImageFromImage(img), nil
}

var _ services.AssetCache = (*AssetCache)(nil)
