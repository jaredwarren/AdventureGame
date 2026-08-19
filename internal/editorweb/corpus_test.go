package editorweb

import (
	"fmt"
	"sort"
	"testing"
)

// corpusStore serves the real assets/maps tree.
func corpusServer(t *testing.T) *Server {
	t.Helper()
	s, err := New(Options{MapsDir: "../../assets/maps", Token: "test-token"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

// knownCorpusIssues pins every validation finding in the shipped maps.
//
// This exists so the corpus can only get cleaner. A new entry means a map
// regressed; a stale entry means someone fixed something and should delete the
// line. Every finding below was verified by hand against the map data.
var knownCorpusIssues = map[string][]string{
	// Two objects share id 5. Persistent pickups and shrines key their save
	// state on "<mapID>:<objectID>", so duplicates share one collected flag.
	// F-5 and field1 are the same map (field1 is stamped onto the F-5 cell), so
	// both report its off-grid sign.
	"field1": {"duplicate_object_id", "sign_not_tile_aligned"},
	"F-5":    {"sign_not_tile_aligned"},

	// F-5 is the authored field1 map, but its neighbours use the generated
	// grid's fixed entry points, which land on wall tiles in the authored
	// layout. Walking through these doors puts the player inside a wall.
	"F-4":  {"door_spawn_in_solid"},
	"F-6":  {"door_spawn_in_solid"},
	"maze": {"door_spawn_in_solid"},

	// Doors authored partly off the map: maze2 door 5 sits at x=-2 and door 7 at
	// y=-8. They still work, because the player collides with the visible part.
	// G-5 is the same map stamped onto its grid cell (south of start).
	"maze2": {"marker_out_of_bounds", "marker_out_of_bounds", "sign_not_tile_aligned"},
	"G-5":   {"sign_not_tile_aligned"},
}

// TestCorpusIssuesAreKnown validates every shipped map and compares the result
// against the pinned list above.
func TestCorpusIssuesAreKnown(t *testing.T) {
	s := corpusServer(t)
	report, err := s.ValidateAll()
	if err != nil {
		t.Fatalf("ValidateAll: %v", err)
	}
	if report.MapCount < 100 {
		t.Fatalf("only %d maps validated; expected the full corpus", report.MapCount)
	}

	got := map[string][]string{}
	for id, issues := range report.Maps {
		for _, i := range issues {
			got[id] = append(got[id], i.Code)
		}
		sort.Strings(got[id])
	}
	for id := range knownCorpusIssues {
		sort.Strings(knownCorpusIssues[id])
	}

	for id, want := range knownCorpusIssues {
		if !equalStrings(got[id], want) {
			t.Errorf("%s: issues %v, pinned %v", id, got[id], want)
		}
	}
	for id, codes := range got {
		if _, known := knownCorpusIssues[id]; !known {
			t.Errorf("%s has new validation issues %v — either fix the map or add it to knownCorpusIssues with a note explaining why it is acceptable", id, codes)
		}
	}
}

// TestCorpusErrorsAreOnlyDuplicateIDs states the remaining error-level debt
// explicitly. Fixing the duplicate ids in field1.tmj and dungeon.tmj (a separate
// data change) should let this assert zero.
func TestCorpusErrorsAreOnlyDuplicateIDs(t *testing.T) {
	s := corpusServer(t)
	report, err := s.ValidateAll()
	if err != nil {
		t.Fatal(err)
	}
	var offenders []string
	for id, issues := range report.Maps {
		for _, i := range issues {
			if i.Severity != SeverityError {
				continue
			}
			if i.Code != "duplicate_object_id" {
				offenders = append(offenders, fmt.Sprintf("%s: %s", id, i.Code))
			}
		}
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Errorf("unexpected error-level issues: %v", offenders)
	}
	if report.ErrorCount != 1 {
		t.Errorf("%d error-level issues, expected the 1 known duplicate object id (field1 x1)", report.ErrorCount)
	}
}

// TestEveryShippedMapLoads is a whole-corpus version of the existing coverage in
// internal/run/grid_maps_test.go, which only exercises 7 of the ~104 maps.
func TestEveryShippedMapLoads(t *testing.T) {
	s := corpusServer(t)
	infos, err := s.Store().List()
	if err != nil {
		t.Fatal(err)
	}
	for _, info := range infos {
		t.Run(info.ID, func(t *testing.T) {
			if info.ParseError != "" {
				t.Fatalf("parse failed: %s", info.ParseError)
			}
			m, _, _, err := s.Store().Read(info.ID)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			for _, i := range Validate(m, info.ID, ValidateOpts{}) {
				if i.Code == "build_failed" {
					t.Errorf("the game's loader rejects this map: %s", i.Message)
				}
			}
		})
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
