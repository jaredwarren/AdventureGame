package pickup

import "fmt"

// PersistentSaveKey returns the save/session key for a Tiled pickup object id.
func PersistentSaveKey(mapID string, tiledObjectID int) string {
	return fmt.Sprintf("%s:%d", mapID, tiledObjectID)
}
