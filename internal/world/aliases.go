// Re-exports keep the stable world package API for callers outside internal/world.
package world

import (
	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world/enemy"
	"github.com/jaredwarren/game-test/internal/world/pickup"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

type TileDef = tile.Tile

func TileDefOf(gid int) TileDef { return tile.DefOf(gid) }
func RegisteredTileGIDs() []int { return tile.RegisteredGIDs() }
func SolidAt(gid, tileIndex int, destroyed map[int]bool, hasKey bool) bool {
	return tile.SolidAt(gid, tileIndex, destroyed, hasKey)
}
func MapTilePersistKey(mapID string, tx, ty int) string {
	return tile.MapTilePersistKey(mapID, tx, ty)
}
func ParseMapTilePersistKey(k string) (string, int, int, bool) {
	return tile.ParseMapTilePersistKey(k)
}

type PickupKind = pickup.Kind

var (
	PickupCoin     = pickup.Coin
	PickupHeart    = pickup.Heart
	PickupBomb     = pickup.Bomb
	PickupSmallKey = pickup.SmallKey
	PickupTorch        = pickup.Torch
	PickupPegasusBoots = pickup.PegasusBoots
	AllPickups         = pickup.All
)

func PickupKindFromTiled(name string) *PickupKind { return pickup.FromTiled(name) }
func PickupKindByID(id int) (*PickupKind, bool)   { return pickup.ByID(id) }
func RegisterPickup(k *PickupKind)                { pickup.Register(k) }
func PersistentPickupSaveKey(mapID string, id int) string {
	return pickup.PersistentSaveKey(mapID, id)
}

type EnemyConfig = enemy.Config

func DefaultEnemyConfig() EnemyConfig { return enemy.DefaultConfig() }
func EnemyConfigFromTiled(o *tiled.Object) EnemyConfig {
	return enemy.ConfigFromTiled(o)
}
func DefaultEnemyTiledProperties() []tiled.Property { return enemy.DefaultTiledProperties() }
func EnemyTiledProperties(cfg EnemyConfig) []tiled.Property {
	return enemy.TiledProperties(cfg)
}

// DefaultEnemyAggroRadiusPx is re-exported for systems that reference the constant.
const DefaultEnemyAggroRadiusPx = enemy.DefaultAggroRadiusPx
