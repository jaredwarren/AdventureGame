// stream.go — on-wire format for recorded input sessions.
//
// Frame uses string names (not bitmasks) so fixture files remain human-
// readable and stable across services.Action enum reordering. The cost
// is a few extra bytes per frame; at 60 Hz with typically <4 actions
// down at once, a one-minute recording is ~100 KB — acceptable for
// tests, acceptable for short debug captures. A future binary format
// can plug in here without rewriting callers: Stream is an interface-
// free value type; only Marshal/Unmarshal are format-specific.
package replay

import (
	"encoding/json"
	"fmt"
	"io"
)

// SchemaVersion is the on-disk format version. Bump on breaking changes
// so future readers can refuse or migrate old streams.
const SchemaVersion = 1

// Frame is one sim tick's worth of recorded input. All collections are
// "currently down" at the moment the tick was sampled; empty slices
// marshal cleanly to null via json omitempty (see struct tags).
//
// Action / Modifier / Mouse names come from the canonical tables in
// internal/input and internal/platform/ebiten (for modifiers/mouse the
// names are duplicated from the services enums — see replay/playback.go
// for the modifier table).
type Frame struct {
	Down      []string `json:"down,omitempty"`
	Modifiers []string `json:"mods,omitempty"`
	Mouse     []string `json:"mouse,omitempty"`
	CursorX   int      `json:"cx,omitempty"`
	CursorY   int      `json:"cy,omitempty"`
}

// Stream is a complete recorded session: metadata + an ordered list of
// per-tick frames. Seed is reserved for future use by simulations that
// introduce RNG on the hot path; today's systems do not, and Seed is
// zero by convention.
type Stream struct {
	Version  int     `json:"version"`
	TickRate int     `json:"tickRate"`
	Seed     int64   `json:"seed,omitempty"`
	Frames   []Frame `json:"frames"`
}

// NewStream constructs an empty Stream with the current schema version
// and the given tick rate. Callers append frames via the Recorder or
// manually in tests.
func NewStream(tickRate int) *Stream {
	return &Stream{
		Version:  SchemaVersion,
		TickRate: tickRate,
		Frames:   make([]Frame, 0, 64),
	}
}

// Append adds a frame to the stream. Primarily for tests; live recording
// goes through Recorder.BeginTick.
func (s *Stream) Append(f Frame) { s.Frames = append(s.Frames, f) }

// Len returns the number of recorded frames. Handy in tests and for
// progress reporting during playback.
func (s *Stream) Len() int {
	if s == nil {
		return 0
	}
	return len(s.Frames)
}

// WriteJSON serializes s to w with 2-space indentation. Errors are
// wrapped so callers can distinguish marshal vs write failures in logs.
func WriteJSON(w io.Writer, s *Stream) error {
	if s == nil {
		return fmt.Errorf("replay: nil stream")
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("replay: marshal: %w", err)
	}
	if _, err := w.Write(append(raw, '\n')); err != nil {
		return fmt.Errorf("replay: write: %w", err)
	}
	return nil
}

// ReadJSON parses a Stream from r. The returned stream is never nil on
// nil error. Callers should verify Version matches their expectation;
// a mismatch is not fatal today (the struct tags tolerate unknown fields
// on future schemas), but it's a useful warning signal.
func ReadJSON(r io.Reader) (*Stream, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("replay: read: %w", err)
	}
	var s Stream
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("replay: unmarshal: %w", err)
	}
	return &s, nil
}
