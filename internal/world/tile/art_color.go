package tile

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
)

// ParseHexColor accepts #rgb, #rrggbb, or #rrggbbaa.
func ParseHexColor(s string) (color.RGBA, error) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "#") {
		return color.RGBA{}, fmt.Errorf("color %q must start with #", s)
	}
	hex := s[1:]
	var r, g, b, a uint8
	a = 0xff
	switch len(hex) {
	case 3:
		ri, err1 := strconv.ParseUint(hex[0:1]+hex[0:1], 16, 8)
		gi, err2 := strconv.ParseUint(hex[1:2]+hex[1:2], 16, 8)
		bi, err3 := strconv.ParseUint(hex[2:3]+hex[2:3], 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return color.RGBA{}, fmt.Errorf("invalid color %q", s)
		}
		r, g, b = uint8(ri), uint8(gi), uint8(bi)
	case 6:
		ri, err1 := strconv.ParseUint(hex[0:2], 16, 8)
		gi, err2 := strconv.ParseUint(hex[2:4], 16, 8)
		bi, err3 := strconv.ParseUint(hex[4:6], 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return color.RGBA{}, fmt.Errorf("invalid color %q", s)
		}
		r, g, b = uint8(ri), uint8(gi), uint8(bi)
	case 8:
		ri, err1 := strconv.ParseUint(hex[0:2], 16, 8)
		gi, err2 := strconv.ParseUint(hex[2:4], 16, 8)
		bi, err3 := strconv.ParseUint(hex[4:6], 16, 8)
		ai, err4 := strconv.ParseUint(hex[6:8], 16, 8)
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			return color.RGBA{}, fmt.Errorf("invalid color %q", s)
		}
		r, g, b, a = uint8(ri), uint8(gi), uint8(bi), uint8(ai)
	default:
		return color.RGBA{}, fmt.Errorf("invalid color %q", s)
	}
	return color.RGBA{R: r, G: g, B: b, A: a}, nil
}

// FormatHexColor formats an opaque or translucent RGBA as #rrggbb or #rrggbbaa.
func FormatHexColor(c color.RGBA) string {
	if c.A == 0xff {
		return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	}
	return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, c.A)
}
