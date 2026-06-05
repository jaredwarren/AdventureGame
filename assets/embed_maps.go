// Package assets holds embedded map data exported from Tiled (JSON .tmj).
// Edit files under maps/ only; a normal `go build` re-embeds them—no copy step.
package assets

import "embed"

//go:embed maps/*.tmj
var MapFS embed.FS
