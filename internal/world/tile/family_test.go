package tile

import (
	"image/color"
	"testing"
)

func TestRegisterFamily_AutoGeneratesAllVariants(t *testing.T) {
	t.Parallel()

	// Register a test family
	testBaseGID := 1000
	testFamilyName := "test_carpet"
	testFill := color.RGBA{0x90, 0x20, 0x40, 0xff}
	testEdge := color.RGBA{0x60, 0x10, 0x20, 0xff}

	RegisterFamily(FamilyConfig{
		Name:        testFamilyName,
		Category:    "decor",
		BaseGID:     testBaseGID,
		Kind:        FamilyFloor,
		FloorWeight: 0.25,
		Style: TileStyle{
			FillColor:   testFill,
			EdgeColor:   testEdge,
			LineWidth:   1.5,
			HasDetail:   true,
			DetailColor: color.RGBA{0xff, 0xd7, 0x00, 0xff},
		},
		Collapsed: true,
	})

	fam, ok := FamilyByName(testFamilyName)
	if !ok {
		t.Fatalf("expected family %q to be found by FamilyByName", testFamilyName)
	}
	if len(fam.GIDs) != int(VariantCount) {
		t.Fatalf("expected %d GIDs in family, got %d", VariantCount, len(fam.GIDs))
	}

	expectedSuffixes := []string{
		"", "_top", "_bottom", "_left", "_right",
		"_ne", "_nw", "_sw", "_se",
		"_ne_inner", "_nw_inner", "_sw_inner", "_se_inner",
	}

	for i, suffix := range expectedSuffixes {
		expectedGID := testBaseGID + i
		expectedName := testFamilyName + suffix

		d := DefOf(expectedGID)
		if d.Name != expectedName {
			t.Errorf("variant %d: expected name %q, got %q", i, expectedName, d.Name)
		}
		if !d.IsFloor() || d.Solid() || d.Wall() || d.Water() {
			t.Errorf("variant %d: expected floor properties, got Solid=%v Wall=%v IsFloor=%v",
				i, d.Solid(), d.Wall(), d.IsFloor())
		}
		if d.SwatchColor != testFill {
			t.Errorf("variant %d: expected swatch %v, got %v", i, testFill, d.SwatchColor)
		}
		if d.VectorDraw == nil {
			t.Errorf("variant %d: expected VectorDraw to be non-nil", i)
		}
		solidRects := SolidRectsAt(expectedGID, 0, nil, false)
		if len(solidRects) != 0 {
			t.Errorf("variant %d: expected empty SolidRects for floor family, got %v", i, solidRects)
		}
	}
}
