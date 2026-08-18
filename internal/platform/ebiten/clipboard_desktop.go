//go:build !js

// clipboard_desktop.go — desktop clipboard backend via github.com/atotto/clipboard.
package ebitenplat

import "github.com/atotto/clipboard"

// Clipboard is the system-clipboard implementation of services.Clipboard.
type Clipboard struct{}

// NewClipboard returns the default desktop clipboard.
func NewClipboard() *Clipboard { return &Clipboard{} }

// WriteText forwards to atotto/clipboard.
func (c *Clipboard) WriteText(text string) error { return clipboard.WriteAll(text) }

// ReadText reads the current clipboard contents.
func (c *Clipboard) ReadText() (string, error) { return clipboard.ReadAll() }
