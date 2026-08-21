package assets

import "embed"

// TileArtFS holds editable vector tile art (*.tile.json).
//
//go:embed tiles/*.tile.json
var TileArtFS embed.FS
