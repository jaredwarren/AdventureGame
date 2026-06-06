package world

import "github.com/jaredwarren/game-test/internal/progression"

// PickupKind describes one collectible item type. Package-level vars such as
// PickupCoin are singleton pointers; Pickup.Kind stores the same pointer.
type PickupKind struct {
	id          int
	tiledName   string
	toast       string
	editorLabel string
	apply       func(*World)
}

// ID is the stable index for atlas frames and save data (matches AllPickups order).
func (k *PickupKind) ID() int {
	if k == nil {
		return 0
	}
	return k.id
}

// TiledName is the "kind" property value in Tiled pickup markers.
func (k *PickupKind) TiledName() string {
	if k == nil {
		return ""
	}
	return k.tiledName
}

// ToastMessage is shown when the player collects this pickup from a chest.
func (k *PickupKind) ToastMessage() string {
	if k == nil || k.toast == "" {
		return "Found Item!"
	}
	return k.toast
}

// EditorLabel is the human-readable name in the map editor item menu.
func (k *PickupKind) EditorLabel() string {
	if k == nil {
		return ""
	}
	return k.editorLabel
}

// ApplyReward applies the gameplay effect of collecting this pickup.
func (k *PickupKind) ApplyReward(w *World) {
	if k == nil || k.apply == nil {
		return
	}
	k.apply(w)
}

var (
	PickupCoin = &PickupKind{
		id:          0,
		tiledName:   "coin",
		toast:       "Found Gold Coin!",
		editorLabel: "Coin",
		apply:       applyCoinPickup,
	}
	PickupHeart = &PickupKind{
		id:          1,
		tiledName:   "heart",
		toast:       "Found Heart!",
		editorLabel: "Heart",
		apply:       applyHeartPickup,
	}
	PickupBomb = &PickupKind{
		id:          2,
		tiledName:   "bomb",
		toast:       "Found Bomb!",
		editorLabel: "Bomb",
		apply:       applyBombPickup,
	}
	PickupSmallKey = &PickupKind{
		id:          3,
		tiledName:   "key",
		toast:       "Found Small Key!",
		editorLabel: "Key",
		apply:       applySmallKeyPickup,
	}
	PickupTorch = &PickupKind{
		id:          4,
		tiledName:   "torch",
		toast:       "Found Torch!",
		editorLabel: "Torch",
		apply:       applyTorchPickup,
	}
)

// AllPickups lists every pickup kind in atlas / save ID order.
var AllPickups = []*PickupKind{
	PickupCoin,
	PickupHeart,
	PickupBomb,
	PickupSmallKey,
	PickupTorch,
}

var (
	pickupByTiledName map[string]*PickupKind
	pickupByID        map[int]*PickupKind
)

func init() {
	rebuildPickupIndexes()
}

func rebuildPickupIndexes() {
	pickupByTiledName = make(map[string]*PickupKind, len(AllPickups))
	pickupByID = make(map[int]*PickupKind, len(AllPickups))
	for _, k := range AllPickups {
		pickupByTiledName[k.tiledName] = k
		pickupByID[k.id] = k
	}
}

// PickupKindFromTiled resolves a Tiled pickup "kind" property. Unknown names
// fall back to PickupCoin.
func PickupKindFromTiled(name string) *PickupKind {
	if k, ok := pickupByTiledName[name]; ok {
		return k
	}
	return PickupCoin
}

// PickupKindByID returns the pickup for a stable atlas/save ID.
func PickupKindByID(id int) (*PickupKind, bool) {
	k, ok := pickupByID[id]
	return k, ok
}

// RegisterPickup adds or replaces a pickup kind (mainly for tests).
func RegisterPickup(k *PickupKind) {
	if k == nil {
		return
	}
	for i, existing := range AllPickups {
		if existing.id == k.id || existing.tiledName == k.tiledName {
			AllPickups[i] = k
			rebuildPickupIndexes()
			return
		}
	}
	AllPickups = append(AllPickups, k)
	rebuildPickupIndexes()
}

func unregisterPickup(k *PickupKind) {
	if k == nil {
		return
	}
	filtered := AllPickups[:0]
	for _, p := range AllPickups {
		if p != k {
			filtered = append(filtered, p)
		}
	}
	AllPickups = filtered
	rebuildPickupIndexes()
}

func applyCoinPickup(w *World) {
	w.Currency += progression.DefaultEconomy().CoinPickupValue
}

func applyHeartPickup(w *World) {
	heal := progression.DefaultEconomy().HeartPickupHeal
	if heal <= 0 || w.HP >= w.MaxHP() {
		return
	}
	w.HP += heal
	if w.HP > w.MaxHP() {
		w.HP = w.MaxHP()
	}
}

func applyBombPickup(w *World) {
	w.Bombs = w.ClampBombsCarry(w.Bombs + 1)
}

func applySmallKeyPickup(w *World) {
	w.SmallKey++
}

func applyTorchPickup(w *World) {
	w.HasTorch = true
}
