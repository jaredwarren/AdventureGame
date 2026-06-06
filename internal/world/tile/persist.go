package tile

import (
	"fmt"
	"strconv"
	"strings"
)

// MapTilePersistKey encodes a map tile coordinate for save/session persistence.
// Format: map_id:tile_x:tile_y.
func MapTilePersistKey(mapID string, tx, ty int) string {
	return fmt.Sprintf("%s:%d:%d", mapID, tx, ty)
}

// ParseMapTilePersistKey decodes MapTilePersistKey.
func ParseMapTilePersistKey(k string) (mapID string, tx, ty int, ok bool) {
	k = strings.TrimSpace(k)
	parts := strings.Split(k, ":")
	if len(parts) < 3 {
		return "", 0, 0, false
	}
	tx, errx := strconv.Atoi(parts[len(parts)-2])
	ty, erry := strconv.Atoi(parts[len(parts)-1])
	if errx != nil || erry != nil {
		return "", 0, 0, false
	}
	mapID = strings.Join(parts[:len(parts)-2], ":")
	if mapID == "" {
		return "", 0, 0, false
	}
	return mapID, tx, ty, true
}
