package services

// Audio is a minimal one-shot SFX surface. Long-running music is a Phase 2
// concern and lives on a separate Music interface when added.
//
// ClipID is a logical identifier (typically a short filename like "hit.wav"
// or a symbolic name once a manifest is introduced). The concrete
// implementation under internal/platform/ebiten knows how to resolve it to
// embedded bytes.
type Audio interface {
	Play(clipID string, volume float64)
	// Tick is called once per sim frame so the backend can prune finished
	// players, respect volume changes, etc. Exposed to keep the platform
	// impl free of hidden goroutines.
	Tick()
}
