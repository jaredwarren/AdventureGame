package editorweb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaredwarren/game-test/internal/tiled"
)

const testToken = "test-token"

// newTestServer builds a server over a scratch maps tree containing F-5 and
// rooms/boss, so the slash-in-id routing can be exercised.
func newTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "rooms"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"F-5", "rooms/boss"} {
		m := validMap()
		if err := writeFixture(root, id, m); err != nil {
			t.Fatal(err)
		}
	}
	s, err := New(Options{MapsDir: root, Token: testToken})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return s, ts
}

func writeFixture(root, id string, m *tiled.Map) error {
	b, err := m.Encode()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, filepath.FromSlash(id)+".tmj"), b, 0o644)
}

// do issues a request with the editor token attached.
func do(t *testing.T, ts *httptest.Server, method, path string, body any) *http.Response {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, ts.URL+path, rdr)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Editor-Token", testToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func decode[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	var out T
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode %s: %v", resp.Request.URL.Path, err)
	}
	return out
}

// TestRoutes pins the route table, including the two things that are easy to get
// wrong: ids containing a slash, and exact-vs-wildcard precedence.
func TestRoutes(t *testing.T) {
	_, ts := newTestServer(t)

	tests := []struct {
		method, path string
		body         any
		want         int
		why          string
	}{
		{"GET", "/api/health", nil, 200, ""},
		{"GET", "/api/schema", nil, 200, ""},
		{"GET", "/api/maps", nil, 200, "the exact literal must beat the {id...} wildcard"},
		{"GET", "/api/maps/F-5", nil, 200, ""},
		{"GET", "/api/maps/rooms/boss", nil, 200, "{id...} must match an embedded slash"},
		{"GET", "/api/maps/F-5.tmj", nil, 200, "the .tmj suffix is accepted and stripped"},
		{"GET", "/api/maps/nope", nil, 404, ""},
		{"GET", "/api/thumbs/rooms/boss", nil, 200, ""},
		{"GET", "/api/validate", nil, 200, ""},
		{"DELETE", "/api/maps/F-5", nil, 405, "deleting maps is not offered"},
		{"POST", "/api/maps", nil, 405, ""},
		{"GET", "/", nil, 200, ""},
		{"GET", "/nonsense", nil, 404, "the catch-all must not serve the app for unknown paths"},
	}
	for _, tc := range tests {
		name := tc.method + " " + tc.path
		t.Run(name, func(t *testing.T) {
			resp := do(t, ts, tc.method, tc.path, tc.body)
			if resp.StatusCode != tc.want {
				t.Errorf("status %d, want %d %s", resp.StatusCode, tc.want, tc.why)
			}
		})
	}
}

// TestStaticDoesNotServeSource makes sure the embedded file server cannot be
// walked out of its subtree.
func TestStaticDoesNotServeSource(t *testing.T) {
	_, ts := newTestServer(t)
	for _, p := range []string{"/static/../store.go", "/static/../../server.go", "/static/js/../../store.go"} {
		resp := do(t, ts, "GET", p, nil)
		body, _ := readAll(resp)
		if resp.StatusCode == 200 && strings.Contains(body, "package editorweb") {
			t.Errorf("%s served Go source", p)
		}
	}
}

// TestSecurityMiddleware covers the local-web-server attack classes: another
// user on the box, DNS rebinding, and a cross-origin form post.
func TestSecurityMiddleware(t *testing.T) {
	_, ts := newTestServer(t)

	t.Run("missing token is rejected", func(t *testing.T) {
		resp, err := ts.Client().Get(ts.URL + "/api/schema")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("status %d, want 401", resp.StatusCode)
		}
	})

	t.Run("wrong token is rejected", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/schema", nil)
		req.Header.Set("X-Editor-Token", "not-the-token")
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("status %d, want 401", resp.StatusCode)
		}
	})

	t.Run("token in the query string works for the first load", func(t *testing.T) {
		resp, err := ts.Client().Get(ts.URL + "/api/schema?t=" + testToken)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status %d, want 200", resp.StatusCode)
		}
	})

	t.Run("the index page needs no token", func(t *testing.T) {
		resp, err := ts.Client().Get(ts.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status %d, want 200; the page is what hands out the token", resp.StatusCode)
		}
	})

	t.Run("a rebinding Host header is rejected", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/schema", nil)
		req.Header.Set("X-Editor-Token", testToken)
		req.Host = "evil.example.com"
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("status %d, want 403", resp.StatusCode)
		}
	})

	t.Run("a cross-origin request is rejected", func(t *testing.T) {
		req, _ := http.NewRequest("PUT", ts.URL+"/api/maps/F-5", bytes.NewReader([]byte("{}")))
		req.Header.Set("X-Editor-Token", testToken)
		req.Header.Set("Origin", "https://evil.example.com")
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("status %d, want 403", resp.StatusCode)
		}
	})

	t.Run("no CORS headers are emitted", func(t *testing.T) {
		resp := do(t, ts, "GET", "/api/schema", nil)
		if v := resp.Header.Get("Access-Control-Allow-Origin"); v != "" {
			t.Errorf("Access-Control-Allow-Origin is %q; this server must never be reachable cross-origin", v)
		}
	})
}

// TestSchemaETag verifies the schema is cacheable, since it is the largest
// payload and never changes within a process.
func TestSchemaETag(t *testing.T) {
	_, ts := newTestServer(t)
	resp := do(t, ts, "GET", "/api/schema", nil)
	etag := resp.Header.Get("ETag")
	if etag == "" {
		t.Fatal("no ETag on the schema response")
	}

	req, _ := http.NewRequest("GET", ts.URL+"/api/schema", nil)
	req.Header.Set("X-Editor-Token", testToken)
	req.Header.Set("If-None-Match", etag)
	resp2, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotModified {
		t.Errorf("status %d, want 304", resp2.StatusCode)
	}
}

// TestMapLoadSaveRoundTrip is the API-level version of the safety property that
// matters most: load a map, save it back untouched, get identical bytes.
func TestMapLoadSaveRoundTrip(t *testing.T) {
	s, ts := newTestServer(t)
	path, _ := s.Store().Path("F-5")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	loaded := decode[mapPayload](t, do(t, ts, "GET", "/api/maps/F-5", nil))
	if loaded.ETag == "" {
		t.Fatal("no etag on load")
	}
	if loaded.Reformats {
		t.Error("a freshly encoded fixture should not report that saving reformats it")
	}

	resp := do(t, ts, "PUT", "/api/maps/F-5", saveRequest{BaseETag: loaded.ETag, Map: loaded.Map})
	if resp.StatusCode != http.StatusOK {
		body, _ := readAll(resp)
		t.Fatalf("save failed: %d %s", resp.StatusCode, body)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Error("a no-op save changed the file; the editor is not safe to leave open")
	}
}

// TestSaveRejectsStaleWrite is the conflict path that fires when the in-game
// editor has written the same file.
func TestSaveRejectsStaleWrite(t *testing.T) {
	s, ts := newTestServer(t)
	loaded := decode[mapPayload](t, do(t, ts, "GET", "/api/maps/F-5", nil))

	// Someone else edits the file.
	other := validMap()
	other.Layers[0].Data[0] = 2
	if _, err := s.Store().Write("F-5", other, ""); err != nil {
		t.Fatal(err)
	}

	resp := do(t, ts, "PUT", "/api/maps/F-5", saveRequest{BaseETag: loaded.ETag, Map: loaded.Map})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status %d, want 409", resp.StatusCode)
	}
	e := decode[apiError](t, resp)
	if e.Kind != "stale_etag" {
		t.Errorf("kind %q, want stale_etag", e.Kind)
	}
	// The UI needs the server's copy to offer reload-vs-overwrite.
	detail, _ := e.Detail.(map[string]any)
	if detail["serverMap"] == nil {
		t.Error("409 did not include the server's current map, so the conflict dialog has nothing to show")
	}

	// Forcing succeeds.
	resp = do(t, ts, "PUT", "/api/maps/F-5", saveRequest{BaseETag: loaded.ETag, Map: loaded.Map, Force: true})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("forced save status %d, want 200", resp.StatusCode)
	}
}

// TestSaveRejectsInvalidMap proves validation gates the write, so a map that
// cannot load in game cannot be written by accident.
func TestSaveRejectsInvalidMap(t *testing.T) {
	s, ts := newTestServer(t)
	loaded := decode[mapPayload](t, do(t, ts, "GET", "/api/maps/F-5", nil))
	path, _ := s.Store().Path("F-5")
	before, _ := os.ReadFile(path)

	loaded.Map.Layers[0].Data[0] = 9999 // unregistered gid: an invisible wall

	resp := do(t, ts, "PUT", "/api/maps/F-5", saveRequest{BaseETag: loaded.ETag, Map: loaded.Map})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", resp.StatusCode)
	}
	if e := decode[apiError](t, resp); e.Kind != "invalid_map" {
		t.Errorf("kind %q, want invalid_map", e.Kind)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Error("a rejected save still modified the file")
	}

	// force writes anyway, because sometimes you are mid-edit.
	resp = do(t, ts, "PUT", "/api/maps/F-5", saveRequest{BaseETag: loaded.ETag, Map: loaded.Map, Force: true})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("forced save status %d, want 200", resp.StatusCode)
	}
}

// TestBadMapIDsAreRejectedOverHTTP checks traversal defense end to end, through
// Go's own path cleaning as well as ours.
func TestBadMapIDsAreRejectedOverHTTP(t *testing.T) {
	s, ts := newTestServer(t)

	// A canary outside the maps root that must never be reachable or writable.
	outside := filepath.Join(filepath.Dir(s.Store().Root), "outside.tmj")
	if err := os.WriteFile(outside, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{
		"/api/maps/..%2f..%2foutside",
		"/api/maps/%2e%2e/outside",
		"/api/maps/rooms/..%2f..%2foutside",
		"/api/maps/a/b/c/d",
	} {
		t.Run(p, func(t *testing.T) {
			resp := do(t, ts, "GET", p, nil)
			if resp.StatusCode == http.StatusOK {
				t.Errorf("status 200; the path escaped the maps root")
			}
		})
	}
}

// TestMarkerEndpoint checks the authoritative marker-creation path over HTTP,
// including that door placement is snapped by internal/world.
func TestMarkerEndpoint(t *testing.T) {
	_, ts := newTestServer(t)

	resp := do(t, ts, "POST", "/api/marker", markerRequest{
		Type: "door", X: 133.5, Y: 97.25, NextObjectID: 7, Name: "north",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	out := decode[struct {
		Object       tiled.Object `json:"object"`
		NextObjectID int          `json:"nextObjectId"`
		HitRect      rect         `json:"hitRect"`
	}](t, resp)

	if out.Object.X != 128 || out.Object.Y != 96 {
		t.Errorf("door placed at (%v,%v), want it snapped to (128,96)", out.Object.X, out.Object.Y)
	}
	if out.Object.Width != 16 || out.Object.Height != 32 {
		t.Errorf("door size %vx%v, want 16x32", out.Object.Width, out.Object.Height)
	}
	if out.NextObjectID != 8 {
		t.Errorf("nextObjectId %d, want 8", out.NextObjectID)
	}
	if out.HitRect.W != 16 || out.HitRect.H != 32 {
		t.Errorf("hit rect %+v, want 16x32", out.HitRect)
	}

	if resp := do(t, ts, "POST", "/api/marker", markerRequest{Type: "nonsense"}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("unknown marker type status %d, want 400", resp.StatusCode)
	}
}

// TestValidateEndpointUsesTheRequestBody proves you can validate unsaved work.
func TestValidateEndpointUsesTheRequestBody(t *testing.T) {
	_, ts := newTestServer(t)
	m := validMap()
	m.Layers[0].Data[0] = 9999

	resp := do(t, ts, "POST", "/api/validate", validateRequest{ID: "F-5", Map: m})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	out := decode[struct {
		Issues     []Issue `json:"issues"`
		ErrorCount int     `json:"errorCount"`
	}](t, resp)
	if out.ErrorCount == 0 || !hasCode(out.Issues, "unknown_gid") {
		t.Errorf("expected unknown_gid, got %v", codes(out.Issues))
	}
}

// TestThumbEndpoint checks the RLE miniature used by the grid browser.
func TestThumbEndpoint(t *testing.T) {
	_, ts := newTestServer(t)
	th := decode[Thumb](t, do(t, ts, "GET", "/api/thumbs/F-5", nil))

	if th.W != 4 || th.H != 3 {
		t.Errorf("thumb is %dx%d, want 4x3", th.W, th.H)
	}
	// The fixture is uniform grass, so it must collapse to a single run.
	if len(th.RLE) != 2 || th.RLE[0] != 1 || th.RLE[1] != 12 {
		t.Errorf("rle %v, want [1 12] for a uniform 4x3 grass map", th.RLE)
	}
	total := 0
	for i := 1; i < len(th.RLE); i += 2 {
		total += th.RLE[i]
	}
	if total != th.W*th.H {
		t.Errorf("rle covers %d tiles, want %d", total, th.W*th.H)
	}
	if len(th.Markers) != 1 {
		t.Errorf("%d markers, want the fixture's single spawn", len(th.Markers))
	}
}

// TestUnknownRequestFieldsAreRejected keeps a client typo from being silently
// ignored — a "froce: true" that does nothing is worse than a 400.
func TestUnknownRequestFieldsAreRejected(t *testing.T) {
	_, ts := newTestServer(t)
	resp := do(t, ts, "POST", "/api/marker", map[string]any{"type": "spawn", "wat": 1})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status %d, want 400 for an unknown request field", resp.StatusCode)
	}
}

func readAll(resp *http.Response) (string, error) {
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			if err.Error() == "EOF" {
				return sb.String(), nil
			}
			return sb.String(), err
		}
	}
}

var _ = fmt.Sprintf
