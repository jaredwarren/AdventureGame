package assets

import "embed"

//go:embed sounds/*.wav
var SoundFS embed.FS
