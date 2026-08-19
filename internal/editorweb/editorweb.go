// Package editorweb serves a local web-based Tiled map editor for assets/maps.
//
// # Architecture contract
//
// The browser owns the document: it holds the parsed .tmj in memory, edits it
// locally (instant paint, client-side undo/redo), and PUTs the whole map on
// save. This server is stateless apart from disk I/O.
//
// # The round-trip rule (load-bearing)
//
// On PUT, unmarshal the client's map into a tiled.Map and write m.Encode().
// NEVER write the client's bytes through. Every .tmj in the corpus uses only
// keys tiled.Map/Layer/Object/Property already model, in struct declaration
// order, so round-tripping is byte-stable today. Re-encoding server-side keeps
// it that way; passing JS's JSON.stringify through would produce gratuitous
// whole-file diffs on every save.
//
// # Deriving, not duplicating
//
// Tile art and marker semantics live in internal/world and must not be
// restated in JavaScript:
//
//   - Tile art is procedural vector code written against the headless
//     tile.Canvas interface. canvasrec.go records those calls into a JSON op
//     list that the browser replays onto a Canvas2D. See canvasrec.go.
//   - Marker hit rects and default properties are *probed* out of
//     world.MarkerObjectHitRect and world.InitMarkerObject rather than
//     reimplemented. See markers.go.
//
// Both derivations are verified at startup and pinned by tests, so a change in
// internal/world either flows through automatically or fails loudly.
//
// This package is deliberately not in archtest's pureCorePackages: it needs
// net/http, embed, and os. It must still never import internal/platform,
// internal/game, or Ebiten.
package editorweb
