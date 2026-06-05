package world

import "fmt"

// PersistentPickupSaveKey returns the save/session key for a Tiled pickup object id.
func PersistentPickupSaveKey(mapID string, tiledObjectID int) string {
	return fmt.Sprintf("%s:%d", mapID, tiledObjectID)
}
