// Package pickup defines collectible item kinds and their rewards.
package pickup

// Kind describes one collectible item type. Package-level vars such as Coin
// are singleton pointers stored on world.Pickup entities.
type Kind struct {
	id          int
	tiledName   string
	toast       string
	editorLabel string
	apply       func(RewardTarget)
}

func (k *Kind) ID() int {
	if k == nil {
		return 0
	}
	return k.id
}

func (k *Kind) TiledName() string {
	if k == nil {
		return ""
	}
	return k.tiledName
}

func (k *Kind) ToastMessage() string {
	if k == nil || k.toast == "" {
		return "Found Item!"
	}
	return k.toast
}

func (k *Kind) EditorLabel() string {
	if k == nil {
		return ""
	}
	return k.editorLabel
}

func (k *Kind) ApplyReward(rt RewardTarget) {
	if k == nil || k.apply == nil {
		return
	}
	k.apply(rt)
}

var (
	Coin = &Kind{
		id: 0, tiledName: "coin", toast: "Found Gold Coin!", editorLabel: "Coin",
		apply: applyCoin,
	}
	Heart = &Kind{
		id: 1, tiledName: "heart", toast: "Found Heart!", editorLabel: "Heart",
		apply: applyHeart,
	}
	Bomb = &Kind{
		id: 2, tiledName: "bomb", toast: "Found Bomb!", editorLabel: "Bomb",
		apply: applyBomb,
	}
	SmallKey = &Kind{
		id: 3, tiledName: "key", toast: "Found Small Key!", editorLabel: "Key",
		apply: applySmallKey,
	}
	Torch = &Kind{
		id: 4, tiledName: "torch", toast: "Found Torch!", editorLabel: "Torch",
		apply: applyTorch,
	}
	PegasusBoots = &Kind{
		id: 5, tiledName: "pegasus_boots", toast: "Found Pegasus Boots!", editorLabel: "Pegasus Boots",
		apply: applyPegasusBoots,
	}
	Shield = &Kind{
		id: 6, tiledName: "shield", toast: "Found Shield!", editorLabel: "Shield",
		apply: applyShield,
	}
)

// All lists every pickup kind in atlas / save ID order.
var All = []*Kind{Coin, Heart, Bomb, SmallKey, Torch, PegasusBoots, Shield}

var (
	byTiledName map[string]*Kind
	byID        map[int]*Kind
)

func init() {
	rebuildIndexes()
}

func rebuildIndexes() {
	byTiledName = make(map[string]*Kind, len(All))
	byID = make(map[int]*Kind, len(All))
	for _, k := range All {
		byTiledName[k.tiledName] = k
		byID[k.id] = k
	}
}

// FromTiled resolves a Tiled pickup "kind" property. Unknown names fall back to Coin.
func FromTiled(name string) *Kind {
	if k, ok := byTiledName[name]; ok {
		return k
	}
	return Coin
}

// ByID returns the pickup for a stable atlas/save ID.
func ByID(id int) (*Kind, bool) {
	k, ok := byID[id]
	return k, ok
}

// Register adds or replaces a pickup kind (mainly for tests).
func Register(k *Kind) {
	if k == nil {
		return
	}
	for i, existing := range All {
		if existing.id == k.id || existing.tiledName == k.tiledName {
			All[i] = k
			rebuildIndexes()
			return
		}
	}
	All = append(All, k)
	rebuildIndexes()
}

func unregister(k *Kind) {
	if k == nil {
		return
	}
	filtered := All[:0]
	for _, p := range All {
		if p != k {
			filtered = append(filtered, p)
		}
	}
	All = filtered
	rebuildIndexes()
}
