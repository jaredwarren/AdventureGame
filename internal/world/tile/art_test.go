package tile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	_, file, _, ok := runtime.Caller(0)
	if ok {
		dir := filepath.Join(filepath.Dir(file), "../../../assets/tiles")
		if err := LoadArtDir(dir); err != nil {
			// Art files are required for spatial-variation tests after migration.
			panic("tile TestMain LoadArtDir: " + err.Error())
		}
	}
	os.Exit(m.Run())
}

func TestArtValidateWeights(t *testing.T) {
	t.Parallel()
	art := Art{
		GID:  1,
		Name: "grass",
		Size: 16,
		Layers: []Shape{
			{ID: "fill", Type: "rect", X: 0, Y: 0, W: 16, H: 16, Fill: "#558855"},
		},
		Spatial: &SpatialGroup{
			Mode: SpatialModeGridHash,
			Variants: []SpatialVariant{
				{ID: "a", Weight: 40, Layers: nil},
				{ID: "b", Weight: 60, Layers: nil},
			},
		},
	}
	if err := art.Validate(); err != nil {
		t.Fatal(err)
	}
	art.Spatial.Variants[1].Weight = 50
	if err := art.Validate(); err == nil {
		t.Fatal("expected weight sum error")
	}
}

func TestArtRoundTripJSON(t *testing.T) {
	t.Parallel()
	a, ok := ArtOf(GIDGrass)
	if !ok {
		t.Fatal("grass art not loaded")
	}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var b Art
	if err := json.Unmarshal(data, &b); err != nil {
		t.Fatal(err)
	}
	if err := b.Validate(); err != nil {
		t.Fatal(err)
	}
	if b.GID != GIDGrass || b.Name != "grass" {
		t.Fatalf("got gid=%d name=%q", b.GID, b.Name)
	}
}

func TestPickSpatialVariant(t *testing.T) {
	t.Parallel()
	vars := []SpatialVariant{
		{ID: "a", Weight: 40},
		{ID: "b", Weight: 30},
		{ID: "c", Weight: 30},
	}
	if PickSpatialVariant(vars, 0) != 0 {
		t.Fatal("roll 0 -> a")
	}
	if PickSpatialVariant(vars, 39) != 0 {
		t.Fatal("roll 39 -> a")
	}
	if PickSpatialVariant(vars, 40) != 1 {
		t.Fatal("roll 40 -> b")
	}
	if PickSpatialVariant(vars, 99) != 2 {
		t.Fatal("roll 99 -> c")
	}
}

func TestParseHexColor(t *testing.T) {
	t.Parallel()
	c, err := ParseHexColor("#558855")
	if err != nil || c.R != 0x55 || c.G != 0x88 || c.B != 0x55 || c.A != 0xff {
		t.Fatalf("got %#v err=%v", c, err)
	}
}

func TestArtJSONKeepsZeroCoords(t *testing.T) {
	t.Parallel()
	art := Art{
		GID:  1,
		Name: "grass",
		Size: 16,
		Layers: []Shape{
			{ID: "fill", Type: "rect", X: 0, Y: 0, W: 16, H: 16, Fill: "#558855"},
		},
	}
	data, err := json.Marshal(art)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, `"x":0`) || !strings.Contains(s, `"y":0`) {
		t.Fatalf("zero coords must not be omitted (browser needs them): %s", s)
	}
}
