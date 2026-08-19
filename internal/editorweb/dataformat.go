package editorweb

import (
	"bytes"
	"regexp"
	"strconv"

	"github.com/jaredwarren/game-test/internal/tiled"
)

// Preserving a file's tile-data layout.
//
// json.MarshalIndent writes every element of a tile layer's "data" array on its
// own line. The generated grid maps are written that way and round-trip
// unchanged, but the hand-authored maps under assets/maps/rooms use one MAP ROW
// per line:
//
//	"data": [
//	  2, 2, 7, 7, 7, 2, 2,
//	  2, 7, 7, 7, 7, 7, 2,
//	  ...
//
// That is a deliberate authoring convention — the array reads as the room's
// floor plan. Flattening it on save would destroy real information for a human
// reader and produce a large diff for a no-op edit.
//
// So the writer preserves whichever layout the file already uses, detected from
// the bytes currently on disk. Nothing here changes the JSON's meaning; it only
// chooses where the newlines go.

// dataLayout is how a file wraps tile-layer data arrays.
type dataLayout int

const (
	layoutOnePerLine dataLayout = iota
	layoutRowPerLine
)

// fileStyle captures the formatting conventions of a map file so a save can
// reproduce them. Both are purely presentational.
type fileStyle struct {
	Data dataLayout
	// InlineProperties writes each Tiled property on one line, the way the
	// hand-authored room prefabs do:
	//   { "name": "socket", "type": "string", "value": "N" }
	InlineProperties bool
}

// detectStyle reads a file's formatting conventions off its bytes.
func detectStyle(raw []byte) fileStyle {
	return fileStyle{
		Data:             detectDataLayout(raw),
		InlineProperties: inlinePropertyRE.Match(raw),
	}
}

// inlinePropertyRE matches a property object already written on a single line.
var inlinePropertyRE = regexp.MustCompile(`\{ "name": .*"value": .* \}`)

// expandedPropertyRE matches the multi-line property object MarshalIndent
// produces. Requiring the value to sit on one line means a property whose value
// is itself an object or array is left alone.
var expandedPropertyRE = regexp.MustCompile(
	`\{\n\s+"name": ([^\n]*),\n\s+"type": ([^\n]*),\n\s+"value": ([^\n]*)\n\s+\}`)

// expandedPropertyNoTypeRE is the same for properties with no "type" (omitempty
// drops it when the type is empty).
var expandedPropertyNoTypeRE = regexp.MustCompile(
	`\{\n\s+"name": ([^\n]*),\n\s+"value": ([^\n]*)\n\s+\}`)

// collapseProperties rewrites expanded property objects onto single lines.
func collapseProperties(encoded []byte) []byte {
	out := expandedPropertyRE.ReplaceAll(encoded, []byte(`{ "name": $1, "type": $2, "value": $3 }`))
	return expandedPropertyNoTypeRE.ReplaceAll(out, []byte(`{ "name": $1, "value": $2 }`))
}

// dataBlockRE matches a "data": [ ... ] array in encoder output. Tile data is
// always a flat list of integers, so this cannot run past the array.
var dataBlockRE = regexp.MustCompile(`(?s)"data": \[(.*?)\]`)

// detectDataLayout reports the convention a file's tile data already uses.
//
// The test is simply whether any line inside the first data array holds more
// than one number.
func detectDataLayout(raw []byte) dataLayout {
	m := dataBlockRE.FindSubmatch(raw)
	if m == nil {
		return layoutOnePerLine
	}
	for _, line := range bytes.Split(m[1], []byte("\n")) {
		if bytes.Count(line, []byte(",")) > 1 {
			return layoutRowPerLine
		}
	}
	return layoutOnePerLine
}

// applyDataLayout rewrites the data arrays in encoded JSON to the given layout.
//
// widths supplies the row width for each data array in document order; a width
// of zero (or a missing entry) leaves that array alone.
func applyDataLayout(encoded []byte, layout dataLayout, widths []int) []byte {
	if layout != layoutRowPerLine {
		return encoded
	}
	i := -1
	return dataBlockRE.ReplaceAllFunc(encoded, func(block []byte) []byte {
		i++
		width := 0
		if i < len(widths) {
			width = widths[i]
		}
		if width <= 0 {
			return block
		}
		inner := dataBlockRE.FindSubmatch(block)[1]
		values := parseInts(inner)
		if len(values) == 0 || len(values)%width != 0 {
			// Not a clean grid (a malformed or partial layer); leave it as is
			// rather than inventing a layout.
			return block
		}
		return wrapRows(values, width, indentOf(inner))
	})
}

// indentOf recovers the element indentation the encoder used, so the rewrapped
// array lines up with the rest of the document.
func indentOf(inner []byte) string {
	for _, line := range bytes.Split(inner, []byte("\n")) {
		trimmed := bytes.TrimLeft(line, " ")
		if len(trimmed) == 0 {
			continue
		}
		return string(line[:len(line)-len(trimmed)])
	}
	return "        "
}

func parseInts(inner []byte) []int {
	fields := bytes.FieldsFunc(inner, func(r rune) bool {
		return r == ',' || r == '\n' || r == ' ' || r == '\t' || r == '\r'
	})
	out := make([]int, 0, len(fields))
	for _, f := range fields {
		n, err := strconv.Atoi(string(f))
		if err != nil {
			return nil
		}
		out = append(out, n)
	}
	return out
}

func wrapRows(values []int, width int, indent string) []byte {
	// The closing bracket sits one indent level out from the elements.
	closeIndent := indent
	if len(closeIndent) >= 2 {
		closeIndent = closeIndent[:len(closeIndent)-2]
	}

	var b bytes.Buffer
	b.WriteString(`"data": [`)
	for i, v := range values {
		if i%width == 0 {
			b.WriteByte('\n')
			b.WriteString(indent)
		} else {
			b.WriteByte(' ')
		}
		b.WriteString(strconv.Itoa(v))
		if i != len(values)-1 {
			b.WriteByte(',')
		}
	}
	b.WriteByte('\n')
	b.WriteString(closeIndent)
	b.WriteByte(']')
	return b.Bytes()
}

// tileLayerWidths lists each tile layer's row width in document order, matching
// the order applyDataLayout walks the data arrays.
func tileLayerWidths(m *tiled.Map) []int {
	var out []int
	for _, l := range m.Layers {
		if l.Type != "tilelayer" {
			continue
		}
		w := l.Width
		if w <= 0 {
			w = m.Width
		}
		out = append(out, w)
	}
	return out
}

// encodeWithStyle encodes a map, reproducing the file's formatting conventions.
func encodeWithStyle(m *tiled.Map, style fileStyle) ([]byte, error) {
	b, err := m.Encode()
	if err != nil {
		return nil, err
	}
	b = applyDataLayout(b, style.Data, tileLayerWidths(m))
	if style.InlineProperties {
		b = collapseProperties(b)
	}
	return b, nil
}
