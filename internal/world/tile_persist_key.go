package world

import (
	"fmt"
	"strconv"
	"strings"
)

// MapTilePersistKey encodes a map tile coordinate for save/session (cracked
// walls opened by bomb, locks opened with a key, etc.).
// Format: map_id:tile_x:tile_y (same convention as GIDAt).
func MapTilePersistKey(mapID string, tx, ty int) string {
	return fmt.Sprintf("%s:%d:%d", mapID, tx, ty)
}

// ParseMapTilePersistKey decodes MapTilePersistKey. Map ids containing ":"
// are supported by treating the last two segments as tx, ty.
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
