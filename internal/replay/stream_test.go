package replay_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/jaredwarren/game-test/internal/replay"
)

// TestStreamRoundtripJSON proves (marshal, unmarshal) is lossless for a
// realistic frame shape. Done against a byte buffer so we don't need
// temp files.
func TestStreamRoundtripJSON(t *testing.T) {
	src := replay.NewStream(60)
	src.Append(replay.Frame{Down: []string{"MoveRight", "Attack"}, CursorX: 160, CursorY: 90})
	src.Append(replay.Frame{Down: []string{"MoveRight"}, Modifiers: []string{"Shift"}})
	src.Append(replay.Frame{Mouse: []string{"Left"}, CursorX: 10, CursorY: 20})

	var buf bytes.Buffer
	if err := replay.WriteJSON(&buf, src); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := replay.ReadJSON(&buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !reflect.DeepEqual(src, got) {
		t.Errorf("roundtrip mismatch\nwant: %+v\n got: %+v", src, got)
	}
}

// TestStreamReadJSONRejectsGarbage confirms we return a typed error on
// malformed input rather than panicking or returning a zero value.
func TestStreamReadJSONRejectsGarbage(t *testing.T) {
	_, err := replay.ReadJSON(bytes.NewBufferString("{not json"))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

// TestWriteJSONRejectsNilStream confirms the guard against nil.
func TestWriteJSONRejectsNilStream(t *testing.T) {
	var buf bytes.Buffer
	if err := replay.WriteJSON(&buf, nil); err == nil {
		t.Fatal("expected error for nil stream")
	}
}

// TestNewStreamStartsEmpty verifies the zero-frames invariant.
func TestNewStreamStartsEmpty(t *testing.T) {
	s := replay.NewStream(60)
	if s.Len() != 0 {
		t.Errorf("Len=%d, want 0", s.Len())
	}
	if s.Version != replay.SchemaVersion {
		t.Errorf("Version=%d, want %d", s.Version, replay.SchemaVersion)
	}
	if s.TickRate != 60 {
		t.Errorf("TickRate=%d, want 60", s.TickRate)
	}
}
