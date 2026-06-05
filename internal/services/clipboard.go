// clipboard.go — system clipboard port.
//
// The clipboard is a platform-specific side channel (XClipboard on Linux,
// NSPasteboard on macOS, OpenClipboard on Windows). Scenes route through
// this port so internal/scenes stays framework-agnostic and easy to run in
// headless tests with a fake clipboard.
//
// Design notes:
//
//   - WriteText / ReadText are both best-effort. On headless CI there may
//     be no clipboard daemon; implementations should return an error in
//     that case rather than panicking. Scenes generally ignore the error
//     (clipboard actions are never gameplay-critical).
//
//   - No binary/image payloads. If we ever need those we'll extend the
//     interface rather than widening WriteText.
package services

// Clipboard is the system clipboard port.
type Clipboard interface {
	// WriteText places text onto the system clipboard. Returns nil on
	// success; returns a non-nil error when the platform clipboard is
	// unavailable (headless CI, sandbox, etc.).
	WriteText(text string) error

	// ReadText returns the current clipboard text, or an error when
	// unavailable. The returned string is empty when the clipboard is
	// empty (that's not an error).
	ReadText() (string, error)
}
