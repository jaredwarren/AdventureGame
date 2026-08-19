package editorweb

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// maxRequestBytes caps request bodies. The largest realistic payload is a whole
// map: the biggest shipped map re-encodes to roughly 40 KB, so 32 MB is orders
// of magnitude of headroom while still bounding memory.
const maxRequestBytes = 32 << 20

// apiError is the single error envelope every handler returns.
//
// Kind is a stable machine-readable discriminator so the client can branch
// (stale_etag opens the conflict dialog, invalid_map opens the validation panel)
// without string-matching on Error.
type apiError struct {
	Error  string `json:"error"`
	Kind   string `json:"kind"`
	Detail any    `json:"detail,omitempty"`
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		// Marshalling our own response failed, so we cannot report it as JSON
		// either. Fall back to plain text rather than emitting a truncated body.
		http.Error(w, "internal error encoding response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_, _ = w.Write(body)
}

func writeErr(w http.ResponseWriter, code int, kind, format string, args ...any) {
	writeJSON(w, code, apiError{Error: fmt.Sprintf(format, args...), Kind: kind})
}

func writeErrDetail(w http.ResponseWriter, code int, kind string, detail any, format string, args ...any) {
	writeJSON(w, code, apiError{Error: fmt.Sprintf(format, args...), Kind: kind, Detail: detail})
}

// readJSON decodes a request body into T with a size cap and strict field
// checking. Unknown fields are rejected so a client typo in a request becomes a
// loud 400 rather than a silently ignored option.
func readJSON[T any](w http.ResponseWriter, r *http.Request, out *T) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return fmt.Errorf("request body exceeds %d bytes", maxRequestBytes)
		}
		return err
	}
	return nil
}
