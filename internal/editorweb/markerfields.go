package editorweb

import (
	"fmt"
	"strings"

	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/pickup"
)

// Property field descriptors.
//
// Fields are derived from each marker's InitMarkerObject template wherever
// possible: the property name, its Tiled type, and its default value all come
// straight out of internal/world. The only hand-written entries are properties
// the game *reads* but InitMarkerObject never *writes* — they cannot be probed
// because nothing emits them. TestMarkerFieldsCoverGameReadProperties pins that
// list against every property key the game actually reads.
//
// TiledType is carried separately from Type because the two genuinely differ:
// door spawn_x is stored as a Tiled "string" and parsed with strconv.ParseFloat
// at load time. The form shows a number input (Numeric), but writes a string
// back, because lying about the on-disk type would break round-tripping.

type fieldOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
	// Sprite indexes the pickup atlas; -1 when the kind has no art.
	Sprite int `json:"sprite,omitempty"`
}

type showIf struct {
	Field string `json:"field"`
	Eq    any    `json:"eq"`
}

type markerField struct {
	Name      string        `json:"name"`
	Label     string        `json:"label"`
	Type      string        `json:"type"`      // int|float|bool|string|multiline|enum|mapref
	TiledType string        `json:"tiledType"` // Property.Type to write back
	Default   any           `json:"default"`
	Widget    string        `json:"widget,omitempty"`
	Unit      string        `json:"unit,omitempty"`
	Min       *float64      `json:"min,omitempty"`
	Max       *float64      `json:"max,omitempty"`
	Step      float64       `json:"step,omitempty"`
	Rows      int           `json:"rows,omitempty"`
	Options   []fieldOption `json:"options,omitempty"`
	Numeric   bool          `json:"numeric,omitempty"`
	Optional  bool          `json:"optional,omitempty"`
	ShowIf    *showIf       `json:"showIf,omitempty"`
	Note      string        `json:"note,omitempty"`
}

func f64(v float64) *float64 { return &v }

// fieldOverride adjusts a derived field for presentation. It never changes the
// property name or the Tiled type — only how the form renders it.
type fieldOverride struct {
	Label   string
	Type    string
	Widget  string
	Unit    string
	Min     *float64
	Max     *float64
	Step    float64
	Rows    int
	Numeric bool
	ShowIf  *showIf
	Note    string
	// Options, when set, is evaluated at schema build time.
	Options func() []fieldOption
}

// fieldOverrides is presentation only. Adding a property to a marker handler in
// Go still surfaces it in the editor without touching this table; an entry here
// just makes the widget nicer.
var fieldOverrides = map[string]map[string]fieldOverride{
	"enemy": {
		"hp":    {Label: "HP", Min: f64(1), Step: 1, Note: "values <= 0 are ignored by enemy.ConfigFromTiled"},
		"speed": {Label: "Speed", Min: f64(0), Step: 0.05},
		"aggro": {Label: "Aggro radius", Unit: "px", Min: f64(0), Step: 4},
		"damage": {
			Label: "Contact damage", Min: f64(1), Step: 1,
			Note: "values <= 0 are ignored by enemy.ConfigFromTiled",
		},
		"is_boss": {Label: "Boss"},
		"is_armored_knight": {
			Label: "Armored knight",
			Note:  "implies armor_hp 3 when armor_hp is absent or <= 0",
		},
		"armor_hp": {
			Label: "Armor HP", Min: f64(0), Step: 1,
			ShowIf: &showIf{Field: "is_armored_knight", Eq: true},
		},
	},
	"pickup": {
		"kind": {Label: "Kind", Type: "enum", Widget: "select", Options: pickupOptions},
	},
	"door": {
		"target_map": {Label: "Target map", Type: "mapref", Widget: "mapref"},
		"spawn_x": {
			Label: "Spawn X", Numeric: true, Step: 1,
			Note: "stored as a Tiled string; the game parses it with strconv.ParseFloat",
		},
		"spawn_y": {
			Label: "Spawn Y", Numeric: true, Step: 1,
			Note: "stored as a Tiled string; the game parses it with strconv.ParseFloat",
		},
	},
	"sign": {
		"text": {
			Label: "Text", Type: "multiline", Widget: "textarea", Rows: 4,
			Note: "signMarker.SpawnFromTiled also stamps GIDSign into the ground layer at the rounded tile",
		},
	},
}

// extraFields are properties the game reads but InitMarkerObject never writes,
// so they cannot be derived from a template. Every entry names the reader that
// justifies it. TestMarkerFieldsCoverGameReadProperties keeps this honest.
var extraFields = map[string][]markerField{
	"pickup": {
		{
			Name: "persistent", Label: "Persistent", Type: "bool", TiledType: "bool",
			Default: false, Widget: "checkbox", Optional: true,
			Note: "read by pickupMarker.SpawnFromTiled; needs a unique object id (save key \"<mapID>:<objectID>\")",
		},
		{
			Name: "chest", Label: "In chest", Type: "bool", TiledType: "bool",
			Default: false, Widget: "checkbox", Optional: true,
			Note: "read by pickupMarker.SpawnFromTiled; implies persistent",
		},
	},
	"door": {
		{
			Name: "spawn_anchor", Label: "Spawn anchor", Type: "enum", TiledType: "string",
			Default: "feet", Widget: "select", Optional: true,
			Options: []fieldOption{
				{Value: "feet", Label: "Feet (default)"},
				{Value: "topleft", Label: "Hitbox top-left"},
			},
			Note: "read by doorSpawnStyleFromTiled; also accepts top_left, top-left, origin",
		},
	},
}

// pickupOptions builds the pickup enum from the registry, so registering a new
// Kind in Go adds it to the dropdown with no edit here.
//
// The pickup atlas has fewer frames than there are kinds (shield has no art), so
// Sprite is -1 for kinds the client must fall back to a glyph for.
func pickupOptions() []fieldOption {
	out := make([]fieldOption, 0, len(pickup.All))
	for _, k := range pickup.All {
		out = append(out, fieldOption{Value: k.TiledName(), Label: k.EditorLabel(), Sprite: k.ID()})
	}
	return out
}

// deriveFields turns a marker template's properties into form fields, then
// applies presentation overrides and appends the un-probeable extras.
func deriveFields(typ string, tmpl tiled.Object) []markerField {
	overrides := fieldOverrides[typ]
	fields := make([]markerField, 0, len(tmpl.Properties)+2)

	for _, p := range tmpl.Properties {
		f := markerField{
			Name:      p.Name,
			Label:     humanize(p.Name),
			Type:      fieldTypeFor(p),
			TiledType: p.Type,
			Default:   p.Value,
		}
		f.Widget = defaultWidget(f.Type)
		applyOverride(&f, overrides[p.Name])
		fields = append(fields, f)
	}

	for _, extra := range extraFields[typ] {
		fields = append(fields, extra)
	}
	return fields
}

// fieldTypeFor maps a Tiled property type onto a form field type.
func fieldTypeFor(p tiled.Property) string {
	switch p.Type {
	case "int":
		return "int"
	case "float":
		return "float"
	case "bool":
		return "bool"
	case "string", "":
		return "string"
	case "color", "file", "object":
		return p.Type
	default:
		// Unknown Tiled types round-trip untouched through a raw text input, so
		// a property this editor has never seen is preserved rather than eaten.
		return "string"
	}
}

func defaultWidget(fieldType string) string {
	switch fieldType {
	case "int", "float":
		return "number"
	case "bool":
		return "checkbox"
	case "multiline":
		return "textarea"
	case "enum":
		return "select"
	default:
		return "text"
	}
}

func applyOverride(f *markerField, o fieldOverride) {
	if o.Label != "" {
		f.Label = o.Label
	}
	if o.Type != "" {
		f.Type = o.Type
		f.Widget = defaultWidget(o.Type)
	}
	if o.Widget != "" {
		f.Widget = o.Widget
	}
	if o.Options != nil {
		f.Options = o.Options()
	}
	f.Unit = orString(o.Unit, f.Unit)
	f.Note = orString(o.Note, f.Note)
	if o.Min != nil {
		f.Min = o.Min
	}
	if o.Max != nil {
		f.Max = o.Max
	}
	if o.Step != 0 {
		f.Step = o.Step
	}
	if o.Rows != 0 {
		f.Rows = o.Rows
	}
	if o.Numeric {
		f.Numeric = true
	}
	if o.ShowIf != nil {
		f.ShowIf = o.ShowIf
	}
}

func orString(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}

// humanize turns a snake_case property name into a form label.
func humanize(name string) string {
	parts := strings.Split(name, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

// markerSchema is one marker type as the browser sees it.
type markerSchema struct {
	Type        string        `json:"type"`
	Label       string        `json:"label"`
	HitRect     hitRectModel  `json:"hitRect"`
	SnapsToGrid bool          `json:"snapsToGrid"`
	Sizable     bool          `json:"sizable"`
	Unique      bool          `json:"unique"`
	Template    tiled.Object  `json:"template"`
	Fields      []markerField `json:"fields"`
}

// sizableTypes are the marker types whose on-disk width/height are meaningful,
// so the editor offers resize handles. Derived: a type is sizable exactly when
// its hit rect tracks the object's own size.
func (m markerSchema) computeSizable() bool {
	return m.HitRect.W.Cw == 1 && m.HitRect.H.Ch == 1
}

// buildMarkerSchemas derives the full marker schema and verifies every hit-rect
// model against internal/world before returning it.
func buildMarkerSchemas() ([]markerSchema, error) {
	names := world.MarkerTypeNames()
	out := make([]markerSchema, 0, len(names))

	for _, typ := range names {
		hr := probeHitRect(typ)
		if err := verifyHitRectModel(typ, hr); err != nil {
			return nil, err
		}
		tmpl, snaps := probeTemplate(typ)

		ms := markerSchema{
			Type:        typ,
			Label:       humanize(typ),
			HitRect:     hr,
			SnapsToGrid: snaps,
			Template:    tmpl,
			Fields:      deriveFields(typ, tmpl),
			// Only one player spawn is meaningful: BuildFromTiled applies each in
			// turn, so the last one wins.
			Unique: typ == "spawn",
		}
		ms.Sizable = ms.computeSizable()
		out = append(out, ms)
	}

	// The unknown-type fallback in world.MarkerObjectHitRect must also hold, or
	// the editor would mislocate objects authored in Tiled that this build does
	// not know about.
	unknown := probeHitRect("\x00not-a-marker-type")
	if err := verifyHitRectModel("\x00not-a-marker-type", unknown); err != nil {
		return nil, fmt.Errorf("unknown marker type fallback: %w", err)
	}
	return out, nil
}

// unknownMarkerHitRect is the model the client uses for object types this build
// does not recognize, matching world.MarkerObjectHitRect's fallback.
func unknownMarkerHitRect() hitRectModel { return probeHitRect("\x00not-a-marker-type") }
