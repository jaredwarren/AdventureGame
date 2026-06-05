// clipboard.go — desktop clipboard backend via github.com/atotto/clipboard.
//
// This file is packaged under platform/ebiten alongside the other desktop
// services (Input/Audio/Assets/Renderer) to keep the platform bundle a
// single import for cmd/game. The dependency (atotto/clipboard) is a pure
// Cgo-free X11/NSPasteboard/Win32 wrapper — unrelated to Ebiten — but co-
// locating it avoids multiplying tiny platform subpackages.
package ebitenplat

import "github.com/atotto/clipboard"

// Clipboard is the system-clipboard implementation of services.Clipboard.
// Zero value is not usable; construct via NewClipboard.
type Clipboard struct{}

// NewClipboard returns the default desktop clipboard. There is no state to
// configure today; the constructor exists so we can add options (e.g. a
// max-length guard or a dry-run mode for tests) without changing callers.
func NewClipboard() *Clipboard { return &Clipboard{} }

// WriteText forwards to atotto/clipboard. Errors surface verbatim so
// callers can distinguish "no clipboard daemon" (Linux headless) from a
// transient failure.
func (c *Clipboard) WriteText(text string) error { return clipboard.WriteAll(text) }

// ReadText reads the current clipboard contents.
func (c *Clipboard) ReadText() (string, error) { return clipboard.ReadAll() }
