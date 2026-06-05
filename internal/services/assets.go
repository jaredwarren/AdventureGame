package services

// Image is an opaque reference to a loaded texture. Platform impls may
// additionally expose a backend-specific accessor (e.g. a *ebiten.Image)
// via a private method; scene code that already imports the backend can
// type-assert, but systems and core packages MUST NOT.
type Image interface {
	// Size returns logical (pre-scale) pixel dimensions.
	Size() (w, h int)
}

// AtlasFrame is a single named/indexed region of a sprite sheet, resolved
// through AssetCache. Dimensions already incorporate per-frame overrides
// declared in the atlas JSON (dw/dh), so callers can draw frames uniformly
// without re-reading atlas metadata.
type AtlasFrame struct {
	Image Image

	// Dst{W,H} is the authored draw size in world pixels. Scenes typically
	// scale the source image onto a DstW×DstH rect at the entity position.
	DstW, DstH float64

	// Offset{X,Y} is an optional draw-origin nudge in world pixels (e.g.
	// pickup.png frames offset a few px so the sprite visually centers on
	// the 12×12 hitbox).
	OffsetX, OffsetY float64

	// Skip is true when the atlas author explicitly marked this frame as
	// absent (e.g. GIDEmpty in the tile atlas). Callers should not draw it.
	Skip bool
}

// Atlas is a fixed-length bag of AtlasFrames indexed by integer ID. The
// ID space matches the atlas JSON order (tile: GID indices; pickup: PickupKind).
type Atlas interface {
	Frame(i int) AtlasFrame
	Count() int
}

// AtlasID identifies the known atlases. Phase 1 uses a closed set; later
// phases may switch to arbitrary string IDs as assets grow.
type AtlasID string

const (
	AtlasTile   AtlasID = "tile"
	AtlasPickup AtlasID = "pickup"
)

// AssetCache is a lazy, idempotent loader for embedded assets. It replaces
// the sync.Once singletons previously held in internal/game.
//
// Methods may block on first call for a given ID (image decode). Subsequent
// calls must be cheap and return the same handle. Thread safety: all methods
// are safe for concurrent use.
type AssetCache interface {
	Atlas(id AtlasID) (Atlas, error)
	MapData(id string) ([]byte, error)
}
