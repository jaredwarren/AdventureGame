// Package sprites holds embedded PNG sprite sheets and their JSON atlases (see docs/sprites.md).
// Import path: github.com/jaredwarren/game-test/assets/sprites.
package sprites

import _ "embed"

// PickupSpriteSheetFile is the PNG filename paired with pickup_atlas.json's "sprite_sheet" field.
// Keep in sync with the //go:embed line below.
const PickupSpriteSheetFile = "legendofzelda_items_sheet.png"

//go:embed legendofzelda_items_sheet.png
var PickupPNG []byte

//go:embed pickup_atlas.json
var PickupAtlasJSON []byte

//go:embed tileset_alpha.png
var TilesetAlphaPNG []byte

// TileSpriteSheetFile is the PNG filename paired with tile_atlas.json's "sprite_sheet" field.
// Keep in sync with the //go:embed line below.
const TileSpriteSheetFile = "overworld-spritesheet.png"

//go:embed overworld-spritesheet.png
var OverworldTilePNG []byte

//go:embed tile_atlas.json
var TileAtlasJSON []byte

//go:embed lake_island_scene.png
var LakeIslandScenePNG []byte

