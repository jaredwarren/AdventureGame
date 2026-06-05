// Package tiled is a tiny JSON reader for orthogonal Tiled maps—enough for this project’s .tmj exports.
//
// Unsupported Tiled features are silently ignored (infinite maps, tile transforms, wang sets, etc.).
// TODO: validate layer dimensions vs Map.Width/Height and surface structured errors for designers.
package tiled

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Map is a subset of Tiled JSON for orthogonal maps with enough fields for lossless round-trip on save.
type Map struct {
	CompressionLevel int             `json:"compressionlevel"`
	Width            int             `json:"width"`
	Height           int             `json:"height"`
	Infinite         bool            `json:"infinite"`
	TileWidth        int             `json:"tilewidth"`
	TileHeight       int             `json:"tileheight"`
	Type             string          `json:"type,omitempty"`
	Version          string          `json:"version,omitempty"`
	TiledVersion     string          `json:"tiledversion,omitempty"`
	Orientation      string          `json:"orientation,omitempty"`
	RenderOrder      string          `json:"renderorder,omitempty"`
	Layers           []Layer         `json:"layers"`
	NextLayerID      int             `json:"nextlayerid,omitempty"`
	NextObjectID     int             `json:"nextobjectid,omitempty"`
	Tilesets         json.RawMessage `json:"tilesets"`
	Properties       []Property      `json:"properties,omitempty"`
}

type Layer struct {
	ID         int        `json:"id,omitempty"`
	Type       string     `json:"type"`
	Name       string     `json:"name,omitempty"`
	Visible    bool       `json:"visible"`
	Opacity    float64    `json:"opacity,omitempty"`
	Width      int        `json:"width,omitempty"`
	Height     int        `json:"height,omitempty"`
	X          float64    `json:"x,omitempty"`
	Y          float64    `json:"y,omitempty"`
	Data       []int      `json:"data,omitempty"`
	Objects    []Object   `json:"objects,omitempty"`
	Properties []Property `json:"properties,omitempty"`
}

type Property struct {
	Name  string `json:"name"`
	Type  string `json:"type,omitempty"`
	Value any    `json:"value"`
}

type Object struct {
	ID         int        `json:"id"`
	Name       string     `json:"name,omitempty"`
	Type       string     `json:"type"`
	X          float64    `json:"x"`
	Y          float64    `json:"y"`
	Width      float64    `json:"width,omitempty"`
	Height     float64    `json:"height,omitempty"`
	Properties []Property `json:"properties,omitempty"`
}

func LoadMap(path string) (*Map, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseMap(b)
}

func ParseMap(b []byte) (*Map, error) {
	var m Map
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	normalizeAfterDecode(&m)
	return &m, nil
}

func normalizeAfterDecode(m *Map) {
	if len(m.Tilesets) == 0 {
		m.Tilesets = json.RawMessage("[]")
	}
	maxOID := 0
	for i := range m.Layers {
		for j := range m.Layers[i].Objects {
			if m.Layers[i].Objects[j].ID > maxOID {
				maxOID = m.Layers[i].Objects[j].ID
			}
		}
	}
	if m.NextObjectID <= maxOID {
		m.NextObjectID = maxOID + 1
	}
}

// Encode returns indented JSON suitable for writing .tmj files.
func (m *Map) Encode() ([]byte, error) {
	cp := *m
	if len(cp.Tilesets) == 0 {
		cp.Tilesets = json.RawMessage("[]")
	}
	return json.MarshalIndent(&cp, "", "  ")
}

// WriteFile writes Encode() output to path (0644).
func (m *Map) WriteFile(path string) error {
	b, err := m.Encode()
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func (m *Map) TileLayer(name string) *Layer {
	for i := range m.Layers {
		if m.Layers[i].Type == "tilelayer" && m.Layers[i].Name == name {
			return &m.Layers[i]
		}
	}
	for i := range m.Layers {
		if m.Layers[i].Type == "tilelayer" {
			return &m.Layers[i]
		}
	}
	return nil
}

// ObjectGroupLayer returns the object group layer named name, or nil.
func (m *Map) ObjectGroupLayer(name string) *Layer {
	for i := range m.Layers {
		if m.Layers[i].Type != "objectgroup" {
			continue
		}
		if name != "" && m.Layers[i].Name != name {
			continue
		}
		return &m.Layers[i]
	}
	return nil
}

func (m *Map) ObjectGroup(name string) []Object {
	lg := m.ObjectGroupLayer(name)
	if lg == nil {
		return nil
	}
	return lg.Objects
}

func ObjProp(o *Object, key string) (string, bool) {
	for _, p := range o.Properties {
		if p.Name == key {
			switch v := p.Value.(type) {
			case string:
				return v, true
			default:
				return fmt.Sprint(v), true
			}
		}
	}
	return "", false
}

// ObjPropBool reads a Tiled bool or string property ("true"/"1"/"yes").
func ObjPropBool(o *Object, key string) (bool, bool) {
	for _, p := range o.Properties {
		if p.Name != key {
			continue
		}
		switch v := p.Value.(type) {
		case bool:
			return v, true
		case float64:
			return v != 0, true
		case string:
			s := strings.ToLower(strings.TrimSpace(v))
			return s == "true" || s == "1" || s == "yes", true
		default:
			s := strings.ToLower(strings.TrimSpace(fmt.Sprint(v)))
			return s == "true" || s == "1" || s == "yes", true
		}
	}
	return false, false
}

func MapProp(m *Map, key string) (string, bool) {
	for _, p := range m.Properties {
		if p.Name == key {
			switch v := p.Value.(type) {
			case string:
				return v, true
			default:
				return fmt.Sprint(v), true
			}
		}
	}
	return "", false
}

func MapPropFloat(m *Map, key string) (float64, bool) {
	for _, p := range m.Properties {
		if p.Name != key {
			continue
		}
		switch v := p.Value.(type) {
		case float64:
			return v, true
		case int:
			return float64(v), true
		case string:
			var f float64
			if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
				return f, true
			}
		}
	}
	return 0, false
}
