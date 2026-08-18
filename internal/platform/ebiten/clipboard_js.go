//go:build js

package ebitenplat

import "errors"

// Clipboard is a no-op clipboard for WASM/browser builds.
type Clipboard struct{}

// NewClipboard returns a WASM-safe clipboard stub.
func NewClipboard() *Clipboard { return &Clipboard{} }

// WriteText is unavailable in the browser build.
func (c *Clipboard) WriteText(text string) error {
	return errors.New("clipboard unavailable on wasm")
}

// ReadText is unavailable in the browser build.
func (c *Clipboard) ReadText() (string, error) {
	return "", errors.New("clipboard unavailable on wasm")
}
